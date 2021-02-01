package forum

import "net/http"

type Parser interface {
	ParseResponse(r *http.Response, uri string) ([]Post, error)
}
