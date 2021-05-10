package forum

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/blugelabs/query_string"
	"golang.org/x/net/html"
	"html/template"
	"log"
	"os"
	"strings"
	"time"
)

type Indexer struct {
	idx        *bluge.Writer
	reader     *bluge.Reader
	batch      *index.Batch
	count      int
	dupCount   int
	batchCount int
	seen       map[string]bool
}

func NewForumIndex(path string) *Indexer {
	//mapping := bluge.NewIndexMapping()
	//docmap := bluge.NewDocumentMapping()
	// store&index everything by default, except for post.html, which should only be stored
	//storeOnlyMapping := bluge.NewTextFieldMapping()
	//storeOnlyMapping.Index = false // do not index, but do store
	//docmap.AddFieldMappingsAt("html", storeOnlyMapping)
	//mapping.AddDocumentMapping("post", docmap)
	config := bluge.DefaultConfig(path)
	// TODO: expose the number of writer threads to the config and set it to a higher number
	idx, err := bluge.OpenWriter(config)
	if err != nil {
		// index, err = bluge.New(path, mapping)
		//index, err = bluge.NewUsing(path, mapping, scorch.Name, scorch.Name, map[string]interface{}{"numSnapshotsToKeep": 0, "unsafe_batch": true})
		//if err != nil {
		panic(err)
		//}
	}

	reader, err := idx.Reader()
	if err != nil {
		panic(err)
	}

	return &Indexer{idx: idx, reader: reader}
}

func postToDocument(p Post) *bluge.Document {
	d := bluge.NewDocument(p.Id)

	/*
		Url          string `json:"url"`
		ThreadId     int    `json:"threadid"`     // thead identifier, same for all posts in a thread
		PostSeq      int    `json:"threadseq"`    // post identifier, sequential in order within a thread page
		PageSeq      int    `json:"pageseq"`      // page (of thread) id, ordered
		ThreadPostId int64  `json:"threadpostid"` // combined key, uniquely identifies a thread+post id, ordered
		Id           string `json:"id"`           // original post key (identifies a post)
		User         string `json:"user"`
		UserIcon     string `json:"usericon"`
		Hdr          string `json:"hdr"`
		Msg          string `json:"msg"`
		Html         string `json:"html"`
	*/

	// TODO complete this
	d.AddField(newStoredKeywordField("url", p.Url))
	d.AddField(newStoredKeywordField("threadid", p.ThreadId)) // NOTE: we switched to strings at some point.
	d.AddField(bluge.NewNumericField("postseq", float64(p.PostSeq)))
	d.AddField(bluge.NewNumericField("pageseq", float64(p.PageSeq)))
	d.AddField(newStoredSortedKeywordField("threadpostid", p.ThreadPostId))
	d.AddField(newStoredKeywordField("user", p.User))
	d.AddField(newStoredKeywordField("usericon", p.UserIcon))
	d.AddField(newStoredTextField("hdr", p.Hdr))
	d.AddField(newStoredTextField("msg", p.Msg))
	d.AddField(newStoredTextField("html", p.Html))
	// d.AddField(bluge.NewCompositeFieldExcluding("_all", []string{"_id"}))
	return d
}

func newStoredTextField(name string, content string) *bluge.TermField {
	field := bluge.NewTextField(name, content)
	field.FieldOptions |= bluge.Store | bluge.SearchTermPositions
	return field
}

func newStoredKeywordField(name string, content string) *bluge.TermField {
	field := bluge.NewKeywordField(name, content)
	field.FieldOptions |= bluge.Store
	return field
}

func newStoredSortedKeywordField(name string, content string) *bluge.TermField {
	field := bluge.NewKeywordField(name, content)
	field.FieldOptions |= bluge.Store | bluge.Sortable
	return field
}

func (f *Indexer) AddPost(b Post) {
	if f.batch == nil {
		f.batch = bluge.NewBatch()
		f.seen = make(map[string]bool)
		f.batchCount = 0
	}
	f.count++
	// todo use id and b to construct a document!
	_, seen := f.seen[b.Id]
	if seen {
		//log.Println("skipping duplicate post", b.Id)
		f.dupCount++
		return
	}
	f.seen[b.Id] = true
	doc := postToDocument(b)
	f.batch.Insert(doc)
	f.batchCount++
	if f.batchCount >= 10000 {
		log.Println("flushing search index batch (", f.count, "total posts seen", f.dupCount, "duplicates)")
		f.idx.Batch(f.batch)
		f.batch.Reset()
		f.seen = make(map[string]bool)
		f.batchCount = 0
	}
}

func (f *Indexer) Close() {
	if f.batch != nil {
		f.idx.Batch(f.batch)
		f.batch.Reset()
	}
	f.idx.Close()
	log.Println("indexed", f.count, "posts")
}

func (f *Indexer) AddPosts(posts []Post) {
	enc := json.NewEncoder(os.Stdout)
	for _, body := range posts {
		enc.Encode(body)
		// f.AddPost(body)
	}
}

// NOTE: this isn't multi-Unicode-codepoint aware, like specifying skintone or
//       gender of an emoji: https://unicode.org/emoji/charts/full-emoji-modifiers.html
func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

// quick trick to convert a name or string into a random (but consistent) RGB HTML color string in the #000000 format
func makeUniqueColor(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return "#" + hex.EncodeToString(bs[0:3])
}

func (f *Indexer) SearchQueryString(q string) SearchResponse {
	// query := bluge.NewMatchQuery(q)
	query, err := querystr.ParseQueryString(q, querystr.DefaultOptions())
	if err != nil {
		log.Println("failed to parse query string...", query, err)
		// TODO implement error returns
		return SearchResponse{
			Results:     nil,
			TimeSeconds: 0,
			ResultCount: 0,
		}
	}
	return f.Search(query)
}

func (f *Indexer) Stats() {
	cnt, err := f.reader.Count()
	if err != nil {
		fmt.Println("error counting", err)
	}
	fmt.Println("index has", cnt, "records")
}

func (f *Indexer) Search(query bluge.Query) (response SearchResponse) {
	fmt.Println("query string:", query)
	//q := bluge.NewQueryStringQuery(query)

	searchRequest := bluge.NewTopNSearch(100, query).WithStandardAggregations()
	//	searchRequest.Fields = []string{"*"}
	//	searchRequest.Size = 100

	//searchRequest.Sort = search.SortOrder{&search.SortField{
	//	Field:   "threadpostid",
	//	Desc:    true,
	//	Type:    search.SortFieldAsNumber,
	//	Mode:    search.SortFieldDefault,
	//	Missing: search.SortFieldMissingLast,
	//}}
	// TODO check if we have a highlighter
	// searchRequest.Highlight = bluge.NewHighlight() // WithStyle("ansi")
	return f.SearchByRequest(searchRequest)
}

func (f *Indexer) SearchThread(threadId string) (response SearchResponse) {
	//q := bluge.NewQueryStringQuery(query)
	// tid := float64(threadId)
	query := bluge.NewTermQuery(threadId)
	query.SetField("threadid")
	//	fmt.Println("searchThread",query.Match)
	log.Println("running searchThread", threadId)

	searchRequest := bluge.NewTopNSearch(100, query)
	searchRequest.SortBy([]string{"threadpostid"})
	//searchRequest.Sort = search.SortOrder{&search.SortField{
	//	Field:   "threadpostid",
	//	Desc:    false,
	//	Type:    search.SortFieldAsNumber,
	//	Mode:    search.SortFieldDefault,
	//	Missing: search.SortFieldMissingLast,
	//}}
	// searchRequest.Highlight = bluge.NewHighlight() // WithStyle("ansi")
	return f.SearchByRequest(searchRequest)
}

func (f *Indexer) SearchByRequest(searchRequest bluge.SearchRequest) SearchResponse {
	//var reader bluge.Reader // TODO this still needs to be split off, put in the f struct, etc.
	log.Println("running SearchByRequest...")
	searchResult, err := f.reader.Search(context.Background(), searchRequest)
	if err != nil {
		// TODO: do something
		log.Println("failed to run query", searchRequest, err)
		return SearchResponse{}
	}
	// searchResult, _ := f.idx.Search(searchRequest)

	results := make([]SearchResult, 0)

	var resultCount uint64
	var duration time.Duration

	if searchResult.Aggregations() != nil {
		resultCount = searchResult.Aggregations().Count()
		duration = searchResult.Aggregations().Duration()
	}

	match, err := searchResult.Next()
	for err == nil && match != nil {
		var r SearchResult
		log.Println("got a search result, examining fields")
		err = match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_id" {
				fmt.Printf("match: %s\n", string(value))
			}
			fmt.Println(field, string(value))
			switch field {
			case "msg":
				r.Msg = string(value)
				r.Html = template.HTML(html.EscapeString(string(value)))
			case "_id":
				r.Id = string(value)
			case "threadid":
				r.ThreadId = string(value)
			case "hdr":
				r.Hdr = string(value)
			case "url":
				r.Url = string(value)
			case "user":
				r.User = string(value)
			}
			return true
		})
		if err != nil {
			log.Fatalf("error loading stored fields: %v", err)
		}
		r.Initials = substr(strings.TrimSpace(r.User), 0, 2)
		r.UserColor = makeUniqueColor(r.User)

		results = append(results, r)
		match, err = searchResult.Next()
	}
	if err != nil {
		log.Fatalf("error iterator document matches: %v", err)
	}
	//
	//fmt.Println("search took", searchResult.Took)
	//fmt.Println(searchResult.Total, "documents found")

	// TODO this entire logic should be ported to the bluge framework up there ^^ in the loop
	/*
		for _, i := range searchResult.Hits {
			//if nr == 0 {
			//	fmt.Println(i.Fields)
			//}
			bytes, err := json.Marshal(i.Fields)
			if err != nil {

			}
			var post SearchResult
			err = json.Unmarshal(bytes, &post)
			if err != nil {

			}
			post.Initials = substr(strings.TrimSpace(post.User), 0, 2)
			post.UserColor = makeUniqueColor(post.User)
			post.Highlights = make(map[string]template.HTML)
			for fieldname, fragment := range i.Fragments {
				// TODO: this should not be indexed, only stored, but something is wrong so we have to filter it here
				if fieldname == "html" {
					continue
				}
				post.Highlights[fieldname] = template.HTML(strings.Join(fragment, " &hellip; "))
			}
			//fmt.Println(nr, i.ID, i.Fragments, i.Fields["html"])
			//fmt.Println(post)
			results = append(results, post)
		}
	*/
	return SearchResponse{
		Results:     results,
		TimeSeconds: duration,    // TODO compute this ourselves? searchResult.Took.Seconds(),
		ResultCount: resultCount, // searchResult.Total,
	}
}
