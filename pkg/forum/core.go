package forum

import "html/template"

type Post struct {
	Url          string `json:"url"`
	ThreadId     int    `json:"threadid"`     // thead identifier, same for all posts in a thread
	PostSeq      int    `json:"threadseq"`    // post identifier, sequential in order within a thread
	ThreadPostId int64  `json:"threadpostid"` // combined key, uniquely identifies a thread+post id
	Id           string `json:"id"`           // original post key (identifies a post)
	User         string `json:"user"`
	UserIcon     string `json:"usericon"`
	Hdr          string `json:"hdr"`
	Msg          string `json:"msg"`
	Html         string `json:"html"`
}
type SearchResponse struct {
	Results     []SearchResult
	TimeSeconds float64
	ResultCount uint64
}

type SearchResult struct {
	Url          string                   `json:"url"`
	ThreadId     int                      `json:"threadid"`     // thead identifier, same for all posts in a thread
	PostSeq      int                      `json:"threadseq"`    // post identifier, sequential in order within a thread
	ThreadPostId int64                    `json:"threadpostid"` // combined key, uniquely identifies a thread+post id
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
