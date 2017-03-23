package meta

import (
	"context"
	"errors"

	"github.com/Focinfi/sqs/log"
	"github.com/coreos/etcd/clientv3"
)

// ErrDataNotFound for data not found error
var ErrDataNotFound = errors.New("etcd: data not found")

type etcd struct {
	cli *clientv3.Client
}

func newEtcd() (*etcd, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{defaultEtcdEndpoint},
	})

	if err != nil {
		return nil, err
	}

	return &etcd{
		cli: cli,
	}, nil
}

func (e *etcd) Get(key string) (string, error) {
	res, err := e.cli.Get(context.TODO(), key)
	if err != nil {
		return "", err
	}

	if len(res.Kvs) == 0 {
		return "", ErrDataNotFound
	}

	return string(res.Kvs[0].Value), nil
}

func (e *etcd) Put(key, value string) error {
	log.DB.Infof("etcd SET: %v, %v\n", key, value)
	_, err := e.cli.Put(context.TODO(), key, value)
	return err
}

func (e *etcd) WatchModify(key string, do func()) {
	ch := e.cli.Watch(context.TODO(), key)

	for {
		resp := <-ch
		log.DB.Infoln("etcd watch:", string(resp.Events[0].Kv.Value))
		if resp.Canceled {
			ch = e.cli.Watch(context.TODO(), key)
			continue
		}

		if err := resp.Err(); err != nil {
			log.DB.Infof("etcd: failed to watch '%s', err: %s.", key, err)
			ch = e.cli.Watch(context.TODO(), key)
			continue
		}

		for _, event := range resp.Events {
			if event.IsModify() {
				do()
			}
		}
	}
}
