package forum

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"log"
)

type Indexer struct {
	idx        *bluge.Writer
	batch      *index.Batch
	count      int
	batchCount int
}

func NewForumIndex(path string) *Indexer {
	//mapping := bluge.NewIndexMapping()
	//docmap := bluge.NewDocumentMapping()
	// store&index everything by default, except for post.html, which should only be stored
	//storeOnlyMapping := bluge.NewTextFieldMapping()
	//storeOnlyMapping.Index = false // do not index, but do store
	//docmap.AddFieldMappingsAt("html", storeOnlyMapping)
	//mapping.AddDocumentMapping("post", docmap)
	index, err := bluge.OpenWriter(bluge.DefaultConfig(path))
	if err != nil {
		// index, err = bluge.New(path, mapping)
		//index, err = bluge.NewUsing(path, mapping, scorch.Name, scorch.Name, map[string]interface{}{"numSnapshotsToKeep": 0, "unsafe_batch": true})
		//if err != nil {
		panic(err)
		//}
	}
	return &Indexer{idx: index}
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
	d.AddField(bluge.NewTextField("url", p.Url))
	d.AddField(bluge.NewNumericField("url", float64(p.ThreadId))) // NOTE: we have to switch to strings at some point.
	d.AddField(bluge.NewTextField("hdr", p.Hdr))
	d.AddField(bluge.NewTextField("hdr", p.Hdr))
	d.AddField(bluge.NewTextField("hdr", p.Hdr))

	return d
}

func (f *Indexer) AddPost(id string, b Post) {
	if f.batch == nil {
		f.batch = bluge.NewBatch()
		f.batchCount = 0
	}
	f.count++
	// todo use id and b to construct a document!
	doc := postToDocument(b)
	f.batch.Insert(doc)
	if f.batchCount > 10000 {
		fmt.Println("flushing search index batch (", f.count, "total posts seen)")
		f.idx.Batch(f.batch)
		f.batch = nil
		f.batchCount = 0
	}
}

func (f *Indexer) Close() {
	if f.batch != nil {
		f.idx.Batch(f.batch)
		f.batch = nil
	}
	f.idx.Close()
	fmt.Println("indexed", f.count, "posts")
}

func (f *Indexer) AddPosts(posts []Post) {
	for _, body := range posts {
		f.AddPost(body.Id, body)
		//  f.idx.Index("test", body)
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
	query := bluge.NewMatchQuery(q)
	return f.Search(query)
}

func (f *Indexer) Search(query bluge.Query) (response SearchResponse) {
	fmt.Println("query string:", query)
	//q := bluge.NewQueryStringQuery(query)

	searchRequest := bluge.NewTopNSearch(100, query)
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

func (f *Indexer) SearchThread(threadId int) (response SearchResponse) {
	//q := bluge.NewQueryStringQuery(query)
	tid := float64(threadId)
	query := bluge.NewNumericRangeInclusiveQuery(tid, tid, true, true)
	query.SetField("threadid")
	//	fmt.Println("searchThread",query.Match)

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
	var reader bluge.Reader // TODO this still needs to be split off, put in the f struct, etc.

	searchResult, err := reader.Search(context.Background(), searchRequest)
	if err != nil {
		// TODO: do something
		return SearchResponse{}
	}
	// searchResult, _ := f.idx.Search(searchRequest)

	results := make([]SearchResult, 0)

	match, err := searchResult.Next()
	for err == nil && match != nil {
		err = match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_id" {
				fmt.Printf("match: %s\n", string(value))
			}
			return true
		})
		if err != nil {
			log.Fatalf("error loading stored fields: %v", err)
		}
		match, err = searchResult.Next()
	}
	if err != nil {
		log.Fatalf("error iterator document matches: %v", err)
	}

	// fmt.Println("search took", searchResult.Took)
	// fmt.Println(searchResult.Total, "documents found")

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
		TimeSeconds: 0, // TODO compute this ourselves? searchResult.Took.Seconds(),
		ResultCount: 0, // searchResult.Total,
	}
}
