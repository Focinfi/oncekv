package mock

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HostOfURL must get the host from the given url
func HostOfURL(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(fmt.Errorf("wrong url format, err: %v", err))
	}

	return u.Host
}

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

// MakeHTTPGetter makes a HTTPGetter using the given params
func MakeHTTPGetter(url string, response string, err error, delay time.Duration) HTTPGetter {
	return HTTPGetterFunc(func(url string) (*http.Response, error) {
		respChan := make(chan *http.Response)

		go func() {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(response)),
			}

			time.AfterFunc(delay, func() {
				respChan <- resp
			})
		}()

		return <-respChan, err
	})
}

// MakeHTTPPoster makes a HTTPPoster  using the given params
func MakeHTTPPoster(url string, response string, err error, delay time.Duration) HTTPPoster {
	return HTTPPosterFunc(func(url string, contentType string, body io.Reader) (*http.Response, error) {
		respChan := make(chan *http.Response)

		go func() {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(response)),
			}

			time.AfterFunc(delay, func() {
				respChan <- resp
			})
		}()

		return <-respChan, err
	})
}

// HTTPGetterCluster combine getters into one getter
func HTTPGetterCluster(getterMap map[string]HTTPGetter) HTTPGetter {
	return HTTPGetterFunc(func(rawurl string) (*http.Response, error) {
		host := HostOfURL(rawurl)
		getter, ok := getterMap[host]
		if !ok {
			panic(fmt.Sprintf("client: no getter can handle %s", host))
		}

		return getter.Get(host)
	})
}

// HTTPPosterCluster combine postter into one postter
func HTTPPosterCluster(posterMap map[string]HTTPPoster) HTTPPoster {
	return HTTPPosterFunc(func(rawurl string, contentType string, body io.Reader) (*http.Response, error) {
		host := HostOfURL(rawurl)
		poster, ok := posterMap[host]
		if !ok {
			panic(fmt.Sprintf("client: no poster can handle %s", host))
		}

		return poster.Post(host, contentType, body)
	})
}
