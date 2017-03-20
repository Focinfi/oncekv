package client

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func mockHTTPGetter(url string, response string, err error, delay time.Duration) httpGetter {
	return httpGetterFunc(func(url string) (*http.Response, error) {
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

func mockHTTPPoster(url string, response string, err error, delay time.Duration) httpPoster {
	return httpPosterFunc(func(url string, contentType string, body io.Reader) (*http.Response, error) {
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
