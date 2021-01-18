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

func NewForumIndex() *Indexer {
	mapping := bleve.NewIndexMapping()
	index, err := bleve.New("example.bleve", mapping)
	if err != nil {
		panic(err)
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
		fmt.Println("flushing search index batch of size", f.batch.TotalDocsSize())
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
