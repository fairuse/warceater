package forum

import (
	"io"
	"net/http"
)

type Parser interface {
	ParseResponse(body io.ReadCloser, header http.Header, uri string) ([]Post, error)
}
