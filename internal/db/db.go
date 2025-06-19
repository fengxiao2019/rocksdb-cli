package db

import (
	"errors"

	"github.com/linxGnu/grocksdb"
)

type KeyValueDB interface {
	GetCF(cf, key string) (string, error)
	PutCF(cf, key, value string) error
	PrefixScanCF(cf, prefix string, limit int) (map[string]string, error)
	ListCFs() ([]string, error)
	CreateCF(cf string) error
	DropCF(cf string) error
	Close()
}

type DB struct {
	db        *grocksdb.DB
	cfHandles map[string]*grocksdb.ColumnFamilyHandle
	ro        *grocksdb.ReadOptions
	wo        *grocksdb.WriteOptions
}

func Open(path string) (*DB, error) {
	cfNames, err := grocksdb.ListColumnFamilies(grocksdb.NewDefaultOptions(), path)
	if err != nil || len(cfNames) == 0 {
		cfNames = []string{"default"}
	}
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	cfOpts := make([]*grocksdb.Options, len(cfNames))
	for i := range cfNames {
		cfOpts[i] = grocksdb.NewDefaultOptions()
	}
	db, cfHandles, err := grocksdb.OpenDbColumnFamilies(opts, path, cfNames, cfOpts)
	if err != nil {
		return nil, err
	}
	cfHandleMap := make(map[string]*grocksdb.ColumnFamilyHandle)
	for i, name := range cfNames {
		cfHandleMap[name] = cfHandles[i]
	}
	return &DB{
		db:        db,
		cfHandles: cfHandleMap,
		ro:        grocksdb.NewDefaultReadOptions(),
		wo:        grocksdb.NewDefaultWriteOptions(),
	}, nil
}

func (d *DB) Close() {
	for _, h := range d.cfHandles {
		h.Destroy()
	}
	d.db.Close()
	d.ro.Destroy()
	d.wo.Destroy()
}

func (d *DB) GetCF(cf, key string) (string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return "", errors.New("column family not found")
	}
	val, err := d.db.GetCF(d.ro, h, []byte(key))
	if err != nil {
		return "", err
	}
	defer val.Free()
	if !val.Exists() {
		return "", errors.New("not found")
	}
	return string(val.Data()), nil
}

func (d *DB) PutCF(cf, key, value string) error {
	h, ok := d.cfHandles[cf]
	if !ok {
		return errors.New("column family not found")
	}
	return d.db.PutCF(d.wo, h, []byte(key), []byte(value))
}

func (d *DB) PrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, errors.New("column family not found")
	}
	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()
	result := make(map[string]string)
	for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
		k := it.Key()
		v := it.Value()
		if !hasPrefix(k.Data(), []byte(prefix)) {
			k.Free()
			v.Free()
			break
		}
		result[string(k.Data())] = string(v.Data())
		k.Free()
		v.Free()
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (d *DB) ListCFs() ([]string, error) {
	return grocksdb.ListColumnFamilies(grocksdb.NewDefaultOptions(), d.db.Name())
}

func (d *DB) CreateCF(cf string) error {
	h, err := d.db.CreateColumnFamily(grocksdb.NewDefaultOptions(), cf)
	if err != nil {
		return err
	}
	d.cfHandles[cf] = h
	return nil
}

func (d *DB) DropCF(cf string) error {
	h, ok := d.cfHandles[cf]
	if !ok {
		return errors.New("column family not found")
	}
	err := d.db.DropColumnFamily(h)
	if err != nil {
		return err
	}
	h.Destroy()
	delete(d.cfHandles, cf)
	return nil
}

func hasPrefix(s, prefix []byte) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := range prefix {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}
