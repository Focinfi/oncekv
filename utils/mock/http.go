package mock

import (
	"io"
	"net/http"
)

// HTTPGetter for mock http.Get
type HTTPGetter interface {
	Get(url string) (resp *http.Response, err error)
}

// HTTPGetterFunc implements HTTPGetter
type HTTPGetterFunc func(url string) (resp *http.Response, err error)

// Get implements HTTPGetter
func (f HTTPGetterFunc) Get(url string) (resp *http.Response, err error) {
	return f(url)
}

// HTTPPoster fot mock http.Post
type HTTPPoster interface {
	Post(url string, contentType string, body io.Reader) (resp *http.Response, err error)
}

// HTTPPosterFunc implements HTTPPoster
type HTTPPosterFunc func(url string, contentType string, body io.Reader) (resp *http.Response, err error)

// Post implements HTTPPoster
func (f HTTPPosterFunc) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	return f(url, contentType, body)
}
