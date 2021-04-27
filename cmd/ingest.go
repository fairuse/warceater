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
	"bytes"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/fairuse/warceater/pkg/forum"
	"github.com/fairuse/warceater/pkg/parsers"
	"github.com/spf13/cobra"
	"github.com/valyala/gozstd"
	"io/ioutil"
	"net/http"

	// "github.com/fairuse/warc"
	"gitlab.roelf.org/warcscan/warcreader/pkg/warcreader"
	"log"
	"os"
	"runtime/pprof"
	"sync"
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

func loadWarc(filename string, parser forum.Parser) {
	workerWaiter := &sync.WaitGroup{}
	indexWaiter := &sync.WaitGroup{}

	bodyStream := make(chan forum.ParserBody)
	postsStream := make(chan []forum.Post)

	fi := forum.NewForumIndex(indexPath)
	defer fi.Close()

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	// wrap the reader into something that produces a progress bar
	stats, err := os.Stat(filename)
	bar := pb.Full.Start64(stats.Size())
	barReader := bufio.NewReader(bar.NewProxyReader(f))

	// create a decompressed version (zlib, zstd, etc.) of the stream
	decomp, err := warcreader.NewDecompressedReader(barReader)
	if err != nil {
		panic(err)
	}

	// create the reader on top of the decompressed stream
	r := warcreader.NewWARCReader(decomp)

	// set up the indexer.
	indexWaiter.Add(1)
	go func() {
		for posts := range postsStream {
			fi.AddPosts(posts)
		}
		indexWaiter.Done()
	}()

	const workerCount = 4
	workerWaiter.Add(workerCount)
	for nr := 0; nr < workerCount; nr++ {
		go func(threadNr int) {
			for body := range bodyStream {
				posts, err := parser.ParseResponse(body.Body, body.Header, body.Uri)
				if err != nil {
					log.Println("[", threadNr, "] failed to interpret response body", err)
					continue
				}
				postsStream <- posts
			}
			workerWaiter.Done()
		}(nr)
	}

	// defer r.Close()
	for {
		record, err := r.NextRecord()
		if err != nil {
			log.Println(err)
			break
		}

		//rectype, ok := record.Headers["content-type"]
		//if !ok {
		//	fmt.Println("content-type not known")
		//}

		// Note: sometimes there is a space before msgtype, sometimes there is not
		if record.Type == warcreader.RecordTypeRequest {
			request, err := http.ReadRequest(bufio.NewReader(record))
			if err != nil {
				log.Println("failed to read request", err)
				continue
			}
			_ = request
			// log.Println("read request",request.URL)
			if request.Method != "GET" {
				fmt.Println("REQ:", request)
				fmt.Println("RECHDR:", record.Headers)
				reqbody, err := ioutil.ReadAll(request.Body)
				if err != nil {
					fmt.Println("failed to read req body")
				}
				fmt.Println("REQBODY:", string(reqbody))
			}
		}
		if record.Type == warcreader.RecordTypeResponse {
			// fmt.Println(record)
			response, err := record.ParseHTTPResponse() // http.ReadResponse(bufio.NewReader(record.Content), nil)
			if err != nil {
				log.Println("failed to read response body", err)
				continue
			}
			uri := record.TargetURI()
			if uri == "" {
				fmt.Println("failed to get warc-target-uri for response", record.Headers)
			}

			// fmt.Println(uri)
			//_, _ = ioutil.ReadAll(response.Body)
			//_ = uri

			solidBody, err := ioutil.ReadAll(response.Body)

			// fmt.Println("RECHDR:",record.Headers)
			body := forum.ParserBody{
				Body:   bytes.NewReader(solidBody),
				Header: response.Header,
				Uri:    uri,
			}
			bodyStream <- body
			//bodies, err := parser.ParseResponse(response.Body, response.Header, uri)
			//if err != nil {
			//	log.Println("failed to interpret response body", err)
			//	continue
			//}
			//fi.AddPosts(bodies)
		}
	}
	close(bodyStream)
	workerWaiter.Wait()
	close(postsStream)
	indexWaiter.Wait()
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
		if cpuProfile != "" {
			f, err := os.Create(cpuProfile)
			if err != nil {
				panic(err)
			}
			pprof.StartCPUProfile(f)
			fmt.Println("enabling CPU profiling, writing to", cpuProfile)
			defer pprof.StopCPUProfile()
		}
		p := parsers.YahooAnswersParser{}
		for _, filename := range args {
			loadWarc(filename, &p)
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
