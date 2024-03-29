package forum

import (
	"html/template"
	"time"
)

type Post struct {
	Url          string `json:"url"`
	ThreadId     string `json:"threadid"`     // thead identifier, same for all posts in a thread
	PostSeq      int    `json:"threadseq"`    // post identifier, sequential in order within a thread page
	PageSeq      int    `json:"pageseq"`      // page (of thread) id, ordered
	ThreadPostId string `json:"threadpostid"` // combined key, uniquely identifies a thread+post id, ordered
	Id           string `json:"id"`           // original post key (identifies a post)
	User         string `json:"user"`
	UserIcon     string `json:"usericon"`
	Hdr          string `json:"hdr"`
	Msg          string `json:"msg"`
	Html         string `json:"html"`
}
type SearchResponse struct {
	Results     []SearchResult
	TimeSeconds time.Duration
	ResultCount uint64
}

type SearchResult struct {
	Url          string                   `json:"url"`
	ThreadId     string                   `json:"threadid"`     // thead identifier, same for all posts in a thread
	PostSeq      int                      `json:"threadseq"`    // post identifier, sequential in order within a thread page
	PageSeq      int                      `json:"pageseq"`      // page (of thread) id, ordered
	ThreadPostId string                   `json:"threadpostid"` // combined key, uniquely identifies a thread+post id
	Id           string                   `json:"id"`
	User         string                   `json:"user"`
	Initials     string                   `json:"initials"`
	UserIcon     string                   `json:"usericon"`
	Hdr          string                   `json:"hdr"`
	Msg          string                   `json:"msg"`
	Html         template.HTML            `json:"html"`
	Highlights   map[string]template.HTML `json:"highlights"`
	UserColor    string                   `json:"usercolor"`
}
