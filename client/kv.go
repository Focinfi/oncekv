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
	requestTimeout = time.Millisecond * 300
	logPrefix      = "oncekv/client:"
	dbPutURLFormat = "%s/key"
)

var (
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

// KV for kv storage
type KV struct {
	cli *Client
}

type kvParams struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewKV returns a new KV
func NewKV() (*KV, error) {
	cli, err := New()
	if err != nil {
		return nil, err
	}

	return &KV{cli: cli}, nil
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

	if duration > config.Config().IdealKVResponseDuration*2 {
		// remove fastDB
		go func() { kv.cli.setFastDB("") }()
	}

	return nil
}

// Delete delete the key
func (kv *KV) Delete(key string) error {
	// TODO: remove old key for save db space
	return nil
}

func (kv *KV) cache(key string) (string, error) {
	url := kv.cli.fastCache
	if url == "" {
		return kv.tryAllCaches(key)
	}

	idealDuration := config.Config().IdealKVResponseDuration
	val, duration, err := kv.find(key, url, idealDuration)
	if err != nil {
		return kv.tryAllCaches(key)
	}

	if val == "" {
		return "", ErrDataNotFound
	}

	// try allCaches to update kv.cli.fastCache
	if duration > idealDuration {
		go func() {
			if _, err := kv.tryAllCaches(key); err != nil {
				log.DB.Error(logPrefix, err)
			}
		}()
	}

	return val, err
}

func (kv *KV) get(key string) (string, error) {
	if kv.cli.fastDB == "" {
		return kv.tryAllDBfind(key)
	}

	val, duration, err := kv.find(key, kv.cli.fastDB, requestTimeout)
	if err != nil {
		log.DB.Error(logPrefix, err)
		return kv.tryAllDBfind(key)
	}

	if duration > config.Config().IdealKVResponseDuration*2 {
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

	for i, db := range dbs {
		go func(index int, url string) {
			val, _, err := kv.find(key, url, requestTimeout)
			if err != nil {
				log.DB.Error(logPrefix, err)
			}

			mux.Lock()
			defer mux.Unlock()
			if val != "" || completeCount == len(dbs) {
				if !got {
					got = true
					fastURL = url

					go func() { data <- val }()
				}
			}
		}(i, db)
	}

	select {
	case <-time.After(requestTimeout):
		return "", ErrTimeout

	case value := <-data:
		log.Biz.Infoln(logPrefix, "end get:", time.Now())

		if fastURL != "" {
			go kv.cli.setFastDB(fastURL)
		}

		if value != "" {
			return value, nil
		}

		return "", ErrDataNotFound
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

	waitTime := config.Config().IdealKVResponseDuration * 6

	select {
	case <-time.After(waitTime):
		return ErrTimeout
	case res := <-result:
		log.Biz.Infoln(logPrefix, "end tryAllDBSet:", time.Now())

		if fastURL != "" {
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
		return requestTimeout, fmt.Errorf("%s failed to set kv, key: %s, value: %v\n", logPrefix, key, value)
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
	case <-time.After(requestTimeout):
		return "", requestTimeout, ErrTimeout

	case err := <-errChan:
		log.DB.Errorln(logPrefix, "find:", err)
		return "", requestTimeout, err

	case res := <-resChan:
		defer res.Body.Close()

		if res.StatusCode == http.StatusOK {
			val, err := kv.parseData(res.Body, key)
			if err == nil {
				return val, time.Now().Sub(begin), nil
			}

			log.Biz.Errorln(logPrefix, "find/parseData error:", err)

			return "", requestTimeout, ErrDataNotFound
		}

		return "", requestTimeout, ErrDataNotFound
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

	for i, cache := range caches {
		go func(index int, url string) {
			val, duration, err := kv.find(key, url, requestTimeout)
			if err != nil {
				log.DB.Error(logPrefix, err)
			}

			mux.Lock()
			defer mux.Unlock()

			if duration <= minDuration {
				minDuration = duration
				fastURL = url
				log.DB.Infoln(fastURL, duration)
			}

			completeCount++
			if completeCount == len(caches) {
				go func() { completed <- true }()

				if !fetched {
					fetched = true
					go func() { data <- val }()
				}
				return
			}

			if val != "" {
				if !fetched {
					fetched = true
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
		if value != "" {
			return value, nil
		}

		return "", ErrDataNotFound
	}
}
