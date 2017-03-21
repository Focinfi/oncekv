package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/oncekv/utils/urlutil"
)

const (
	logPrefix      = "oncekv/client:"
	dbPutURLFormat = "%s/key"
)

var (
	requestTimeout       = config.Config().ClientRequestTimeout
	idealReponseDuration = config.Config().IdealResponseDuration
	// ErrDataNotFound for data not found response
	ErrDataNotFound = fmt.Errorf("%s data not found", logPrefix)

	// ErrTimeout for timeout
	ErrTimeout = fmt.Errorf("%s timeout", logPrefix)
)

type httpGetter interface {
	Get(url string) (resp *http.Response, err error)
}

type httpGetterFunc func(url string) (resp *http.Response, err error)

func (f httpGetterFunc) Get(url string) (resp *http.Response, err error) {
	return f(url)
}

type httpPoster interface {
	Post(url string, contentType string, body io.Reader) (resp *http.Response, err error)
}

type httpPosterFunc func(url string, contentType string, body io.Reader) (resp *http.Response, err error)

func (f httpPosterFunc) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	return f(url, contentType, body)
}

var defaultGetter = httpGetter(httpGetterFunc(http.Get))
var defaultPoster = httpPoster(httpPosterFunc(http.Post))

// Option for Client option
type Option struct {
	RequestTimeout        time.Duration
	IdealResponseDuration time.Duration
}

// KV for kv storage
type KV struct {
	cli    *Client
	option *Option
}

type kvParams struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// DefaultKV returns a new KV with default option
// RequestTimeout: 100ms
// IdealResponseDuration: 50ms
func DefaultKV() (*KV, error) {
	return NewKV(nil)
}

// NewKV returns a new KV
func NewKV(option *Option) (*KV, error) {
	cli, err := New()
	if err != nil {
		return nil, err
	}

	if option == nil {
		option = &Option{
			RequestTimeout:        requestTimeout,
			IdealResponseDuration: idealReponseDuration,
		}
	}

	return &KV{cli: cli, option: option}, nil
}

// Get get the value of the key
func (kv *KV) Get(key string) (string, error) {
	val, err := kv.cache(key)
	log.DB.Infoln(logPrefix, val, err)

	// believe cache, if cache alive, it can always right
	if err == ErrDataNotFound {
		return "", err
	}

	if err == nil {
		return val, nil
	}

	val, err = kv.get(key)
	if err != nil {
		log.DB.Error(logPrefix, err)
		return "", err
	}

	return val, nil
}

// Put put key/value pair
func (kv *KV) Put(key string, value string) error {
	if kv.cli.fastDB == "" {
		return kv.tryAllDBSet(key, value)
	}

	duration, err := kv.set(key, value, kv.cli.fastDB)
	if err != nil {
		log.DB.Error(logPrefix, err)
		return kv.tryAllDBSet(key, value)
	}

	if duration > idealReponseDuration {
		// remove fastDB
		go func() { kv.cli.setFastDB("") }()
	}

	return nil
}

func (kv *KV) cache(key string) (string, error) {
	url := kv.cli.fastCache
	if url == "" {
		return kv.tryAllCaches(key)
	}

	val, _, err := kv.find(key, url, idealReponseDuration)
	if err == ErrDataNotFound {
		return "", err
	}

	if err != nil {
		return kv.tryAllCaches(key)
	}

	return val, err
}

func (kv *KV) get(key string) (string, error) {
	if kv.cli.fastDB == "" {
		return kv.tryAllDBfind(key)
	}

	val, duration, err := kv.find(key, kv.cli.fastDB, requestTimeout)
	if err == ErrDataNotFound {
		return "", err
	}

	if err != nil {
		log.DB.Error(logPrefix, err)
		return kv.tryAllDBfind(key)
	}

	if duration > idealReponseDuration {
		go func() { kv.cli.setFastDB("") }()
	}

	return val, nil
}

func (kv *KV) tryAllDBfind(key string) (string, error) {
	dbs := make([]string, len(kv.cli.dbs))
	copy(dbs, kv.cli.dbs)
	log.Biz.Infoln(logPrefix, "start get:", time.Now(), dbs)
	if len(dbs) == 0 {
		return "", fmt.Errorf("%s databases are not available\n", logPrefix)
	}

	var got bool
	var mux sync.Mutex
	var data = make(chan string)
	var completeCount int
	var fastURL string
	var resErr error

	for i, db := range dbs {
		go func(index int, url string) {
			val, _, err := kv.find(key, url, requestTimeout)
			if err != nil {
				log.DB.Error(logPrefix, err)
			}

			mux.Lock()
			defer mux.Unlock()
			if val != "" || err == ErrDataNotFound || completeCount == len(dbs) {
				if !got {
					got = true
					fastURL = url
					resErr = err

					go func() { data <- val }()
				}
			}
		}(i, db)
	}

	select {
	case <-time.After(requestTimeout):
		go kv.cli.setFastDB("")
		return "", ErrTimeout

	case value := <-data:
		log.Biz.Infoln(logPrefix, "end get:", time.Now())

		if value != "" || resErr == ErrDataNotFound {
			go kv.cli.setFastDB(fastURL)
		}

		return value, resErr
	}
}

func (kv *KV) tryAllDBSet(key string, value string) error {
	dbs := make([]string, len(kv.cli.dbs))
	copy(dbs, kv.cli.dbs)
	log.Biz.Infoln(logPrefix, "start tryAllDBSet:", time.Now(), dbs)
	if len(dbs) == 0 {
		return fmt.Errorf("%s db unavailable", logPrefix)
	}

	var mux sync.Mutex
	var fetched bool
	var fastURL string
	var completeCount int
	var err error

	var result = make(chan error)

	for i, db := range dbs {
		go func(index int, url string) {
			_, err = kv.set(key, value, url)

			if err != nil {
				log.DB.Error(logPrefix, err)
			}

			mux.Lock()
			defer mux.Unlock()

			completeCount++
			if err == nil || completeCount >= len(dbs) {
				if !fetched {
					fetched = true
					fastURL = url
					go func() { result <- err }()
				}
			}
		}(i, db)
	}

	select {
	case <-time.After(requestTimeout):
		go func() { kv.cli.setFastDB("") }()
		return ErrTimeout
	case res := <-result:
		log.Biz.Infoln(logPrefix, "end tryAllDBSet:", time.Now())

		if res == nil {
			go kv.cli.setFastDB(fastURL)
		}

		return res
	}
}

func (kv *KV) set(key string, value string, url string) (time.Duration, error) {
	log.Biz.Debugln(logPrefix, "put: ", key, value, url)
	begin := time.Now()
	b, err := json.Marshal(&kvParams{Key: key, Value: value})
	if err != nil {
		return requestTimeout, err
	}

	res, err := defaultPoster.Post(fmt.Sprintf(dbPutURLFormat, urlutil.MakeURL(url)), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return requestTimeout, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return requestTimeout, fmt.Errorf("%s failed to set kv(url: %s), key: %s, value: %v\n", logPrefix, url, key, value)
	}

	return time.Now().Sub(begin), nil
}

func (kv *KV) parseData(readCloser io.ReadCloser, key string) (string, error) {
	b, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return "", err
	}

	log.DB.Infoln("Message Resp:", string(b))

	param := &kvParams{}
	if err := json.Unmarshal(b, param); err != nil {
		return "", err
	}

	if param.Key != key {
		return "", fmt.Errorf("%s wrong response for key='%s'\n", logPrefix, key)
	}

	if param.Value == "" {
		return "", fmt.Errorf("%s empty value response for key = '%s'\n", logPrefix, key)
	}

	return param.Value, nil
}

func (kv *KV) find(key string, url string, timeout time.Duration) (value string, duration time.Duration, err error) {
	begin := time.Now()
	resChan := make(chan *http.Response)
	errChan := make(chan error)

	go func() {
		res, err := defaultGetter.Get(fmt.Sprintf("%s/key/%s", urlutil.MakeURL(url), key))
		if err != nil {
			errChan <- err
			return
		}

		resChan <- res
	}()

	select {
	case <-time.After(timeout):
		return "", requestTimeout, ErrTimeout

	case err := <-errChan:
		log.DB.Errorln(logPrefix, "find:", err)
		return "", requestTimeout, err

	case res := <-resChan:
		defer res.Body.Close()
		duration = time.Now().Sub(begin)

		if res.StatusCode == http.StatusNoContent {
			return "", duration, ErrDataNotFound
		}

		if res.StatusCode == http.StatusOK {
			val, err := kv.parseData(res.Body, key)
			if err == nil {
				return val, duration, nil
			}

			log.Biz.Errorln(logPrefix, "find/parseData error:", err)
			return "", requestTimeout, err
		}

		return "", requestTimeout, ErrTimeout
	}
}

// try all caching urls, set the fastCache
func (kv *KV) tryAllCaches(key string) (string, error) {
	caches := make([]string, len(kv.cli.caches))
	copy(caches, kv.cli.caches)
	log.Biz.Infoln(logPrefix, "start tryAllCaches:", time.Now(), caches)
	if len(caches) == 0 {
		return "", fmt.Errorf("%s caches are unavailable ", logPrefix)
	}

	var fetched bool
	var mux sync.Mutex
	var data = make(chan string)
	var completeCount int
	var fastURL string
	var minDuration = requestTimeout
	var completed = make(chan bool)
	var resErr error

	for i, cache := range caches {
		go func(index int, url string) {
			val, duration, err := kv.find(key, url, requestTimeout)
			log.DB.Infoln(logPrefix, key, url, val, duration, err)
			if err != nil {
				log.DB.Error(err)
			}

			mux.Lock()
			defer mux.Unlock()

			if duration <= minDuration {
				minDuration = duration
				fastURL = url
				log.DB.Infoln(fastURL, duration)
			}

			completeCount++
			log.DB.Infoln(completeCount, len(caches))
			if completeCount == len(caches) {
				go func() { completed <- true }()

				if !fetched {
					fetched = true
					resErr = err
					go func() { data <- val }()
				}
				return
			}

			if val != "" || err == ErrDataNotFound {
				if !fetched {
					fetched = true
					resErr = err
					go func() { data <- val }()
				}
			}
		}(i, cache)
	}

	go func() {
		<-completed
		if fastURL != "" {
			go kv.cli.setFastCache(fastURL)
		}
	}()

	select {
	case <-time.After(requestTimeout):
		return "", ErrTimeout

	case value := <-data:
		log.Biz.Println(logPrefix, "end tryAllCaches:", time.Now())
		return value, resErr
	}
}
