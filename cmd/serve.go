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
	"fmt"
	"github.com/fairuse/warceater/pkg/forum"
	"html/template"
	"strconv"

	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"net/http"
)

type SearchController struct {
	idx *forum.Indexer
}

func makeUnsafe(s string) template.HTML {
	return template.HTML(s)
}

func (s *SearchController) handleSearch(ctx *gin.Context) {
	ctx.Request.ParseForm()
	for key, value := range ctx.Request.PostForm {
		fmt.Println(key, value)
	}
	queryStr := ctx.Request.PostFormValue("query")
	response := s.idx.SearchQueryString(queryStr)
	ctx.HTML(http.StatusOK, "index", gin.H{
		"title": "WARCeater 0.0",
		"add": func(a int, b int) int {
			return a + b
		},
		"results":     response.Results,
		"makeUnsafe":  makeUnsafe,
		"query":       queryStr,
		"resultCount": response.ResultCount,
		"searchTime":  response.TimeSeconds,
	})
}

func (s *SearchController) handleThread(ctx *gin.Context) {
	threadIdStr := ctx.Param("threadid")
	threadId, err := strconv.Atoi(threadIdStr)
	fmt.Println("handleThread:", threadIdStr, threadId)
	if err != nil {
		return // TODO
	}
	response := s.idx.SearchThread(threadId)
	ctx.HTML(http.StatusOK, "index", gin.H{
		"title": "WARCeater 0.0",
		"add": func(a int, b int) int {
			return a + b
		},
		"results":     response.Results,
		"makeUnsafe":  makeUnsafe,
		"query":       "_THREAD_",
		"resultCount": response.ResultCount,
		"searchTime":  response.TimeSeconds,
	})
}

func serve() {
	fi := forum.NewForumIndex(indexPath)
	defer fi.Close()

	srv := SearchController{idx: fi}

	router := gin.Default()

	router.Use(static.Serve("/", static.LocalFile("./static", false)))

	//new template engine
	router.HTMLRender = ginview.Default()

	router.GET("/", func(ctx *gin.Context) {
		//render with master
		ctx.HTML(http.StatusOK, "index", gin.H{
			"title": "Index title!",
			"add": func(a int, b int) int {
				return a + b
			},
		})
	})

	router.POST("/search", srv.handleSearch)
	router.GET("/thread/:threadid", srv.handleThread)

	//	router.GET("/page", func(ctx *gin.Context) {
	//		//render only file, must full name with extension
	//		ctx.HTML(http.StatusOK, "page.html", gin.H{"title": "Page file title!!"})
	//	})

	router.Run(":9090")
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a web server for browsing the index",
	Long: `"serve" launches a web server that gives an html view on the given index.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("serve called")
		serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
