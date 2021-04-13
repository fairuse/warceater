package forum

import (
	"io"
	"net/http"
)

// wrapper struct to make it easier to package responses
type ParserBody struct {
	Body   io.Reader
	Header http.Header
	Uri    string
}

type Parser interface {
	ParseResponse(body io.Reader, header http.Header, uri string) ([]Post, error)
}
