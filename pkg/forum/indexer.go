package forum

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"html/template"
	"strings"
)

type Indexer struct {
	idx   bleve.Index
	batch *bleve.Batch
	count int
}

func NewForumIndex(path string) *Indexer {
	mapping := bleve.NewIndexMapping()
	docmap := bleve.NewDocumentMapping()
	// store&index everything by default, except for post.html, which should only be stored
	storeOnlyMapping := bleve.NewTextFieldMapping()
	storeOnlyMapping.Index = false // do not index, but do store
	docmap.AddFieldMappingsAt("html", storeOnlyMapping)
	mapping.AddDocumentMapping("post", docmap)
	index, err := bleve.Open(path)
	if err != nil {
		index, err = bleve.New(path, mapping)
		if err != nil {
			panic(err)
		}
	}
	return &Indexer{idx: index}
}

func (f *Indexer) AddPost(id string, b Post) {
	if f.batch == nil {
		f.batch = f.idx.NewBatch()
	}
	f.count++
	f.batch.Index(id, b)
	if f.batch.TotalDocsSize() > 16*1024*1024 {
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
	query := bleve.NewQueryStringQuery(q)
	return f.Search(query)
}

func (f *Indexer) Search(query query.Query) (response SearchResponse) {
	fmt.Println("query string:", query)
	//q := bleve.NewQueryStringQuery(query)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 100
	//searchRequest.Sort = search.SortOrder{&search.SortField{
	//	Field:   "threadpostid",
	//	Desc:    true,
	//	Type:    search.SortFieldAsNumber,
	//	Mode:    search.SortFieldDefault,
	//	Missing: search.SortFieldMissingLast,
	//}}
	searchRequest.Highlight = bleve.NewHighlight() // WithStyle("ansi")
	return f.SearchByRequest(searchRequest)
}

func (f *Indexer) SearchThread(threadId int) (response SearchResponse) {
	//q := bleve.NewQueryStringQuery(query)
	tid := float64(threadId)
	incl := true
	query := bleve.NewNumericRangeInclusiveQuery(&tid, &tid, &incl, &incl)
	query.SetField("threadid")
	//	fmt.Println("searchThread",query.Match)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 100
	searchRequest.Sort = search.SortOrder{&search.SortField{
		Field:   "threadpostid",
		Desc:    false,
		Type:    search.SortFieldAsNumber,
		Mode:    search.SortFieldDefault,
		Missing: search.SortFieldMissingLast,
	}}
	// searchRequest.Highlight = bleve.NewHighlight() // WithStyle("ansi")
	return f.SearchByRequest(searchRequest)
}

func (f *Indexer) SearchByRequest(searchRequest *bleve.SearchRequest) SearchResponse {
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
