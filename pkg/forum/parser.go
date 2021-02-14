package forum

import (
	"io"
	"net/http"
)

// wrapper struct to make it easier to package responses
type ParserBody struct {
	Body   io.ReadCloser
	Header http.Header
	Uri    string
}

type Parser interface {
	ParseResponse(body io.ReadCloser, header http.Header, uri string) ([]Post, error)
}
