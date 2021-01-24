/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"fmt"
	"github.com/CorentinB/warc"
	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/microcosm-cc/bluemonday"
	"github.com/valyala/gozstd"
	"github.com/fairuse/warceater/pkg/forum"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
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

func testBody(r *http.Response, uri string) ([]forum.Body, error) {
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

	bodies := make([]forum.Body, 0)

	doc.Find(".content-border").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		id, _ := s.Attr("id")
		user := s.Find(".post-user").Text()
		msg := s.Find(".post-message").Text()
		hdr := s.Find(".post-header").Text()
		userIconUri, _ := s.Find(".post-avatar").Find(".avatar").Find("img").Attr("src")
		// fmt.Println(userIconUri, ok)

		_ = sanitizer
		ohtml, _ := goquery.OuterHtml(s.Find(".post-message")) // todo handle error
		sanehtml := sanitizer.Sanitize(ohtml)
		if len(msg) > 0 {
			// fmt.Printf("Post %s [%d]: %s : %s - %s\n", id, i, user, len(hdr), len(msg))
			x := forum.Body{Hdr: hdr, Msg: msg, User: user, Id: id, UserIcon: userIconUri, Html: sanehtml}
			bodies = append(bodies, x)
			//fmt.Println("test HTML:", ohtml)
			//fmt.Println("sane HTML:", sanehtml)

		}
	})
	return bodies, nil
}

func testWarc(filename string) {

	fi := forum.NewForumIndex(indexPath)
	defer fi.Close()

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	stats, err := os.Stat(filename)
	bar := pb.Full.Start64(stats.Size())
	barReader := bar.NewProxyReader(f)

	r, err := warc.NewReader(barReader)
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
			fi.TestIndex(bodies)
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
	bar.Finish()
}

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a WARC file and add it to the index",
	Long: `Read any number of WARC files provided on the command line, and ingest them into the
index. Each WARC file may be uncompressed, gzip compressed or bzip2 compressed.
Each WARC file is parsed and all html pages are scanned against the predefined
patters to extract posts. Each post individually is inserted into the search
index. This can be a time-consuming operation.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		for _, filename := range args {
			testWarc(filename)
		}
	},
}

// var inputWarcs []string

func init() {
	rootCmd.AddCommand(ingestCmd)
	// ingestCmd.PersistentFlags().StringArrayVarP(&inputWarcs, "input", "i", []string{}, "input WARC file to read from (.gz and .bz2 supported)")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ingestCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ingestCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
