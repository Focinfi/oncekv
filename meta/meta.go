package meta

const defaultEtcdEndpoint = "localhost:2379"

// KV defines a KV storage
type KV interface {
	Get(key string) (string, error)
	Put(key string, value string) error
}

// ModifyWatcher defines watch one key's modification
type ModifyWatcher interface {
	WatchModify(key string, do func())
}

// Meta for oncekv meta store
type Meta interface {
	KV
	ModifyWatcher
}

// Default for default Meta
var Default Meta

// New returns a kv
func New() (Meta, error) {
	etcd, err := newEtcd()
	if err != nil {
		return nil, err
	}

	return etcd, nil
}

func init() {
	meta, err := New()
	if err != nil {
		panic(err)
	}

	Default = meta
}
