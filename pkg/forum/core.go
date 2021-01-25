package forum

import "html/template"

type Post struct {
	Id       string `json:"id"`
	User     string `json:"user"`
	UserIcon string `json:"usericon"`
	Hdr      string `json:"hdr"`
	Msg      string `json:"msg"`
	Html     string `json:"html"`
}

type SearchResult struct {
	Id         string                   `json:"id"`
	User       string                   `json:"user"`
	UserIcon   string                   `json:"usericon"`
	Hdr        string                   `json:"hdr"`
	Msg        string                   `json:"msg"`
	Html       template.HTML            `json:"html"`
	Highlights map[string]template.HTML `json:"highlights"`
}
