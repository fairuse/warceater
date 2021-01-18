package main

import (
	"bufio"
	"fmt"
	"github.com/CorentinB/warc"
	"github.com/PuerkitoBio/goquery"
	"github.com/blevesearch/bleve/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/valyala/gozstd"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"os"
	"strings"
)

func testDict() {
	var samples [][]byte
	for i := 0; i < 1000; i++ {
		sample := fmt.Sprintf("this is a dict sample number %d", i)
		samples = append(samples, []byte(sample))
	}

	// Build a dictionary with the desired size of 8Kb.
	dict := gozstd.BuildDict(samples, 8*1024)

	// Create CDict from the dict.
	cd, err := gozstd.NewCDict(dict)
	if err != nil {
		log.Fatalf("cannot create CDict: %s", err)
	}
	defer cd.Release()
}

type Body struct {
	Id       string
	User     string
	UserIcon string
	Hdr      string
	Msg      string
}

func testBody(r *http.Response, uri string) ([]Body, error) {
	sanitizer := bluemonday.UGCPolicy()

	ctype := r.Header.Get("content-type")
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("not text/html")
	}

	root, err := html.Parse(r.Body)
	if err != nil {
		log.Println("error parsing response body", err)
		return nil, err
	}

	doc := goquery.NewDocumentFromNode(root) // not sure where to pass URI.. the internal constructor supports it, but it is not available to us

	bodies := make([]Body, 0)

	doc.Find(".content-border").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		id, _ := s.Attr("id")
		user := s.Find(".post-user").Text()
		msg := s.Find(".post-message").Text()
		hdr := s.Find(".post-header").Text()
		userIconUri, _ := s.Find(".post-avatar").Find(".avatar").Find("img").Attr("src")
		// fmt.Println(userIconUri, ok)
		html, _ := goquery.OuterHtml(s.Find(".post-message")) // todo handle error
		sanehtml := sanitizer.Sanitize(html)
		if len(msg) > 0 {
			// fmt.Printf("Post %s [%d]: %s : %s - %s\n", id, i, user, len(hdr), len(msg))
			x := Body{Hdr: hdr, Msg: msg, User: user, Id: id, UserIcon: userIconUri}
			bodies = append(bodies, x)
			fmt.Println("test HTML:", html)
			fmt.Println("sane HTML:",sanehtml)

		}
	})
	return bodies, nil
}

type ForumIndex struct {
	idx   bleve.Index
	batch *bleve.Batch
	count int
}

func newForumIndex() *ForumIndex {
	mapping := bleve.NewIndexMapping()
	index, err := bleve.New("example.bleve", mapping)
	if err != nil {
		panic(err)
	}
	return &ForumIndex{idx: index}
}

func (f *ForumIndex) addBody(id string, b Body) {
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

func (f *ForumIndex) Close() {
	f.idx.Batch(f.batch)
	f.batch = nil
	f.idx.Close()
	fmt.Println("indexed", f.count, "posts")
}

func (f *ForumIndex) testIndex(bodies []Body) {
	for _, body := range bodies {
		f.addBody(body.Id, body)
		//  f.idx.Index("test", body)
	}
}

func testWarc() {

	fi := newForumIndex()
	defer fi.Close()

	f, err := os.Open("/big/pubdata/forums/lol/forums.eune.leagueoflegends.com-00000.warc.gz")
	if err != nil {
		panic(err)
	}
	r, err := warc.NewReader(f)
	if err != nil {
		panic(err)
	}

	defer r.Close()
	for {
		record, err := r.ReadRecord(false)
		if err != nil {
			log.Println(err)
			break
		}

		rectype := record.Header.Get("content-type")

		// Note: sometimes there is a space before msgtype, sometimes there is not
		if rectype == "application/http; msgtype=request" {
			request, err := http.ReadRequest(bufio.NewReader(record.Content))
			if err != nil {
				log.Println("failed to read request", err)
				continue
			}
			_ = request
			// log.Println("read request",request.URL)
		}
		if rectype == "application/http; msgtype=response" {
			// fmt.Println(record)
			response, err := http.ReadResponse(bufio.NewReader(record.Content), nil)
			if err != nil {
				log.Println("failed to read response body", err)
				continue
			}
			uri := record.Header.Get("warc-target-uri")
			bodies, err := testBody(response, uri)
			if err != nil {
				log.Println("failed to interprset response body", err)
				continue
			}
			fi.testIndex(bodies)
			/*
				content, err := ioutil.ReadAll(response.Body)
				response.Body.Close()

				if err != nil {
					log.Println("failed to interpret response body", err, "read",len(content),"bytes, while content was",len(content),"bytes")
					continue
				}
				ctype := response.Header.Get("content-type")
				if strings.HasPrefix(ctype, "text/html") {
					fmt.Println(record.Header.Get("warc-target-uri"), rectype, ctype, len(content) )
					testBody(string(content)) // note, we do not handle any content encoding yet!
				}
				_ = content
			*/
			//println(string(content))
		}
	}
}

func main() {
	testDict()
	testWarc()
}
