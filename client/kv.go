package client

import (
	"bytes"
	"encoding/json"
	"errors"
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

const requestTimeout = time.Millisecond * 300

// ErrDataNotFound for data not found response
var ErrDataNotFound = errors.New("oncekv: data not found")

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

// Get get the value of the key
func (kv *KV) Get(key string) (string, error) {
	val, err := kv.getFromCache(key)
	fmt.Println("getFromCache: ", val, err)
	if err == nil {
		return val, nil
	}

	if err == ErrDataNotFound {
		return "", err
	}

	log.DB.Error(err)

	val, err = kv.getFromDB(key)
	if err != nil {
		return "", err
	}

	return val, nil
}

// Put put key/value pair
func (kv *KV) Put(key string, value string) error {
	dbs := make([]string, len(kv.cli.dbs))
	copy(dbs, kv.cli.dbs)
	log.Biz.Println("Start fetchFromDBs:", time.Now(), dbs)
	if len(dbs) == 0 {
		return errors.New("oncekv: db unavailable")
	}

	var fetched bool
	var mux sync.Mutex
	var pushed = make(chan bool)
	var completeCount int
	var completed = make(chan bool)
	var fastURL string
	var err error

	for i, db := range dbs {
		go func(index int, url string) {
			err = kv.setValue(key, value, url)

			if err != nil {
				log.DB.Error(err)
			}

			mux.Lock()
			defer mux.Unlock()

			if err == nil && !fetched {
				fetched = true
				fastURL = url
				go func() { pushed <- true }()
			}

			completeCount++
			if completeCount >= len(dbs) {
				go func() { completed <- true }()
			}
		}(i, db)
	}

	if fastURL != "" {
		go kv.cli.setFastDB(fastURL)
	}

	waitTime := config.Config().IdealKVResponseDuration * 6

	select {
	case <-time.After(waitTime):
		log.Biz.Println("End fetchFromDB:", time.Now())
		return errors.New("oncekv: db slow")
	case <-completed:
		return err
	case <-pushed:
		log.Biz.Println("End fetchFromDB:", time.Now())
		return nil
	}
}

// Delete delete the key
func (kv *KV) Delete(key string) error {
	// TODO: remove old key for save db space
	return nil
}

func (kv *KV) setValue(key string, value string, url string) error {
	log.Biz.Println("ToSetMessage: ", key, value, url)
	b, err := json.Marshal(&kvParams{Key: key, Value: value})
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("%s/key", urlutil.MakeURL(url)), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("oncekv: failed to set kv, key: %s, value: %v", key, value)
	}

	return nil
}

func (kv *KV) getFromCache(key string) (string, error) {
	url := kv.cli.fastCache
	if url == "" {
		return kv.fetchFromCache(key)
	}

	idealDuration := config.Config().IdealKVResponseDuration
	val, duration, err := kv.fetchValue(key, idealDuration, url)
	if err != nil {
		return kv.fetchFromCache(key)
	}

	if val == "" {
		return "", ErrDataNotFound
	}

	// for update kv.cli.fastCache
	if duration > idealDuration {
		go func() {
			if _, err := kv.fetchFromCache(key); err != nil {
				log.DB.Error(err)
			}
		}()
	}

	return val, err
}

func (kv *KV) getFromDB(key string) (string, error) {
	dbs := make([]string, len(kv.cli.dbs))
	copy(dbs, kv.cli.dbs)
	log.Biz.Println("Start fetchFromDBs:", time.Now(), dbs)
	if len(dbs) == 0 {
		return "", errors.New("oncekv: db unavailable")
	}

	var fetched bool
	var mux sync.Mutex
	var data = make(chan string)
	var completeCount int
	var completed = make(chan bool)
	var fastURL string

	for i, db := range dbs {
		go func(index int, url string) {
			val, _, err := kv.fetchValue(key, requestTimeout, url)
			if err != nil {
				log.DB.Error(err)
			}

			mux.Lock()
			defer mux.Unlock()

			if val != "" && !fetched {
				fetched = true
				fastURL = url
				go func() { data <- val }()
			}

			completeCount++
			if completeCount >= len(dbs) {
				go func() { completed <- true }()
			}
		}(i, db)
	}

	if fastURL != "" {
		go kv.cli.setFastDB(fastURL)
	}

	waitTime := config.Config().IdealKVResponseDuration * 6

	select {
	case <-time.After(waitTime):
		log.Biz.Println("End fetchFromDB:", time.Now())
		return "", errors.New("onckv: db slow")
	case <-completed:
		return "", ErrDataNotFound
	case value := <-data:
		log.Biz.Println("End fetchFromDB:", time.Now())
		return value, nil
	}
}

func (kv *KV) tryParseResponse(readCloser io.ReadCloser, key string) (string, error) {
	defer readCloser.Close()

	b, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return "", err
	}

	fmt.Println("Message Resp:", string(b))

	param := &kvParams{}
	if err := json.Unmarshal(b, param); err != nil {
		return "", err
	}

	if param.Key != key {
		return "", fmt.Errorf("oncekv: wrong response for key='%s'", key)
	}

	if param.Value == "" {
		return "", fmt.Errorf("onckv: empty value response for key = %s", key)
	}

	return param.Value, nil
}

func (kv *KV) fetchValue(key string, timeout time.Duration, url string) (value string, duration time.Duration, err error) {
	begin := time.Now()
	resChan := make(chan *http.Response)
	errChan := make(chan error)

	go func() {
		res, err := http.Get(fmt.Sprintf("%s/key/%s", urlutil.MakeURL(url), key))
		if err != nil {
			errChan <- err
			return
		}

		resChan <- res
	}()

	select {
	case <-time.After(config.Config().IdealKVResponseDuration):
		fmt.Println("fetchValue timeout")
		return "", -1, errors.New("oncekv: timeout")
	case err := <-errChan:
		log.DB.Error("fetchValue:", err)
		return "", -1, err
	case res := <-resChan:
		if res.StatusCode == http.StatusOK {
			val, err := kv.tryParseResponse(res.Body, key)
			log.Biz.Println("fetchValue: ", val, err)
			if err == nil {
				return val, time.Now().Sub(begin), nil
			}
		}
	}

	return "", -1, ErrDataNotFound
}

// try all caching urls, set the fastCache
func (kv *KV) fetchFromCache(key string) (string, error) {
	caches := make([]string, len(kv.cli.caches))
	copy(caches, kv.cli.caches)
	log.Biz.Println("Start fetchFromCache:", time.Now(), caches)
	if len(caches) == 0 {
		return "", errors.New("oncekv: cache unavailable")
	}

	var fetched bool
	var mux sync.Mutex
	var data = make(chan string)
	var completeCount int
	var fastURL string

	for i, cache := range caches {
		go func(index int, url string) {
			val, duration, err := kv.fetchValue(key, requestTimeout, url)
			if err != nil {
				log.DB.Error(err)
			}

			mux.Lock()
			defer mux.Unlock()

			completeCount++
			if completeCount >= len(caches) {
				if val != "" {
					go func() { data <- val }()
					return
				}

				go func() { data <- "" }()
			}

			if val != "" && !fetched {
				fetched = true
				if duration < requestTimeout {
					fastURL = url
				}
				go func() { data <- val }()
			}
		}(i, cache)
	}

	if fastURL != "" {
		go kv.cli.setFastCache(fastURL)
	}

	waitTime := config.Config().IdealKVResponseDuration * 2

	select {
	case <-time.After(waitTime):
		log.Biz.Println("End fetchFromCache:", time.Now())
		return "", errors.New("oncekv: cache slow")
	case value := <-data:
		log.Biz.Println("End fetchFromCache:", time.Now())
		if value != "" {
			return value, nil
		}

		return "", ErrDataNotFound
	}
}

// NewKV returns a new KV
func NewKV() (*KV, error) {
	cli, err := New()
	if err != nil {
		return nil, err
	}

	return &KV{cli: cli}, nil
}
