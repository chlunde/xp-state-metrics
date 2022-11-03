package metrics

import (
	"go.uber.org/multierr"
	"k8s.io/client-go/tools/cache"
)

// TeeStore implements the cache.Store interface and delegates all
// calls to the stores. For read operations, the first store is used.
type TeeStore struct {
	stores []cache.Store
}

var _ cache.Store = &TeeStore{}

func NewTeeStore(stores ...cache.Store) *TeeStore {
	return &TeeStore{stores: stores}
}

func (t *TeeStore) Add(obj interface{}) error {
	var err error
	for _, s := range t.stores {
		multierr.Append(err, s.Add(obj))
	}
	return err
}

func (t *TeeStore) Update(obj interface{}) error {
	var err error
	for _, s := range t.stores {
		multierr.Append(err, s.Update(obj))
	}
	return err
}

func (t *TeeStore) Delete(obj interface{}) error {
	var err error
	for _, s := range t.stores {
		multierr.Append(err, s.Delete(obj))
	}
	return err
}

func (t *TeeStore) List() []interface{} { return t.stores[0].List() }

func (t *TeeStore) ListKeys() []string { return t.stores[0].ListKeys() }

func (t *TeeStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return t.stores[0].Get(obj)
}

func (t *TeeStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return t.stores[0].GetByKey(key)
}

func (t *TeeStore) Replace(list []interface{}, str string) error {
	var err error
	for _, s := range t.stores {
		multierr.Append(err, s.Replace(list, str))
	}
	return err
}

func (t *TeeStore) Resync() error {
	var err error
	for _, s := range t.stores {
		multierr.Append(err, s.Resync())
	}
	return err
}
