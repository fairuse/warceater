package forum

import (
	"fmt"
	"github.com/blevesearch/bleve/v2"
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

func (f *Indexer) AddBody(id string, b Body) {
	if f.batch == nil {
		f.batch = f.idx.NewBatch()
	}
	f.count++
	f.batch.Index(id, b)
	if f.batch.TotalDocsSize() > 100*1024*124 {
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

func (f *Indexer) TestIndex(bodies []Body) {
	for _, body := range bodies {
		f.AddBody(body.Id, body)
		//  f.idx.Index("test", body)
	}
}

func (f *Indexer) Search(query string) {
	fmt.Println("query string:", query)
	q := bleve.NewQueryStringQuery(query)

	searchRequest := bleve.NewSearchRequest(q)
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 100
	searchRequest.Highlight = bleve.NewHighlight() // WithStyle("ansi")
	searchResult, _ := f.idx.Search(searchRequest)

	fmt.Println("search took", searchResult.Took)
	fmt.Println(searchResult.Total, "documents found")

	for nr, i := range searchResult.Hits {
		fmt.Println(nr, i.ID, i.Fragments, i.Fields["html"])
	}
}
