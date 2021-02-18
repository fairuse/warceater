package forum

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/blugelabs/bluge/search"
	//"github.com/blevesearch/bleve/v2"
	//"github.com/blevesearch/bleve/v2/index/scorch"
	//"github.com/blevesearch/bleve/v2/search"
	//"github.com/blevesearch/bleve/v2/search/query"
	"html/template"
	"strings"
)

type Indexer struct {
	idx   *bluge.Writer
	batch *index.Batch
	count int
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

func postToDocument(p Post) bluge.Document {
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
	d.AddField(bluge.NewNumericField("url", float64(p.ThreadId)))// NOTE: we have to switch to strings at some point.
	d.AddField(bluge.NewTextField("hdr", p.Hdr))
	d.AddField(bluge.NewTextField("hdr", p.Hdr))
	d.AddField(bluge.NewTextField("hdr", p.Hdr))

	return d
}

func (f *Indexer) AddPost(id string, b Post) {
	if f.batch == nil {
		f.batch = bluge.NewBatch()
	}
	f.count++
	// todo use id and b to construct a document!
	f.batch.Insert(b)
	if f.batch. > 16*1024*1024 {
		fmt.Println("flushing search index batch of size", f.batch.TotalDocsSize(), "(", f.count, "total posts seen)")
		f.idx.Batch(f.batch)
		f.batch = nil
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
	query := bluge.NewQueryStringQuery(q)
	return f.Search(query)
}

func (f *Indexer) Search(query query.Query) (response SearchResponse) {
	fmt.Println("query string:", query)
	//q := bluge.NewQueryStringQuery(query)

	searchRequest := bluge.NewSearchRequest(query)
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 100
	//searchRequest.Sort = search.SortOrder{&search.SortField{
	//	Field:   "threadpostid",
	//	Desc:    true,
	//	Type:    search.SortFieldAsNumber,
	//	Mode:    search.SortFieldDefault,
	//	Missing: search.SortFieldMissingLast,
	//}}
	searchRequest.Highlight = bluge.NewHighlight() // WithStyle("ansi")
	return f.SearchByRequest(searchRequest)
}

func (f *Indexer) SearchThread(threadId int) (response SearchResponse) {
	//q := bluge.NewQueryStringQuery(query)
	tid := float64(threadId)
	incl := true
	query := bluge.NewNumericRangeInclusiveQuery(&tid, &tid, &incl, &incl)
	query.SetField("threadid")
	//	fmt.Println("searchThread",query.Match)

	searchRequest := bluge.NewSearchRequest(query)
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 100
	searchRequest.Sort = search.SortOrder{&search.SortField{
		Field:   "threadpostid",
		Desc:    false,
		Type:    search.SortFieldAsNumber,
		Mode:    search.SortFieldDefault,
		Missing: search.SortFieldMissingLast,
	}}
	// searchRequest.Highlight = bluge.NewHighlight() // WithStyle("ansi")
	return f.SearchByRequest(searchRequest)
}

func (f *Indexer) SearchByRequest(searchRequest *bluge.SearchRequest) SearchResponse {
	searchResult, _ := f.idx.Search(searchRequest)

	fmt.Println("search took", searchResult.Took)
	fmt.Println(searchResult.Total, "documents found")

	results := make([]SearchResult, 0)

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
	return SearchResponse{
		Results:     results,
		TimeSeconds: searchResult.Took.Seconds(),
		ResultCount: searchResult.Total,
	}
}
