package main

import (
	"bufio"
	"fmt"
	"github.com/CorentinB/warc"
	"github.com/valyala/gozstd"
	"github.com/PuerkitoBio/goquery"
	"github.com/blevesearch/bleve/v2"
	"log"
	"net/http"
	"golang.org/x/net/html"
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
   User string
   Hdr string
   Msg string
}

func testBody(r *http.Response, uri string) ([]Body, error) {
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

	bodies := make([]Body,0)

	doc.Find(".content-border").Each(func(i int, s *goquery.Selection) {
	    // For each item found, get the band and title
	    user := s.Find(".post-user").Text()
	    msg := s.Find(".post-message").Text()
	    hdr := s.Find(".post-header").Text()
            userIconUri, ok := s.Find(".post-avatar").Find(".avatar").Find("img").Attr("src")
            fmt.Println(userIconUri,ok)
            if len(msg)>0 {
		fmt.Printf("Post %d: %s : %s - %s\n", i, user, len(hdr), len(msg))
		x := Body{ Hdr: hdr, Msg: msg, User: user }
		bodies = append(bodies, x)
		
	    }
	  })
	return bodies, nil
}

type ForumIndex struct {
	idx bleve.Index
}

func newForumIndex() *ForumIndex {
	mapping := bleve.NewIndexMapping()
	index, err := bleve.New("example.bleve", mapping)
	if err != nil {
		panic(err)
	}
	return &ForumIndex{ idx: index }
}

func (f* ForumIndex) testIndex(bodies []Body) {
	for _, body := range bodies {
		f.idx.Index("test",body)
	}
}

func testWarc() {

	fi := newForumIndex()

	f, err := os.Open("forums.eune.leagueoflegends.com-00000.warc.gz")
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
				log.Println("failed to read request",err)
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

