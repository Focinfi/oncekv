package master

import (
	"context"
	"fmt"

	"github.com/coreos/etcd/clientv3"
)

const defaultEtcdEndpoint = "localhost:2379"

type kv interface {
	Get(key string) (string, error)
	Set(key string, value string) error
}

type etcdKV struct {
	cli *clientv3.Client
}

func (kv *etcdKV) Get(key string) (string, error) {
	res, err := kv.cli.Get(context.TODO(), key)
	if err != nil {
		return "", err
	}

	if len(res.Kvs) == 0 {
		return "", err
	}

	return string(res.Kvs[0].Value), nil
}

func (kv *etcdKV) Set(key, value string) error {
	fmt.Printf("SET: %v, %v\n", key, value)
	_, err := kv.cli.Put(context.TODO(), key, value)
	return err
}

func newEtcdKV() (*etcdKV, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{defaultEtcdEndpoint},
	})

	if err != nil {
		return nil, err
	}

	return &etcdKV{
		cli: cli,
	}, nil
}
