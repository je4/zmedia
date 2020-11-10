package database

import (
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/goph/emperror"
	"github.com/je4/zmedia/v2/pkg/filesystem"
)

const (
	ESTATE_PREFIX     = "est-"
	STORAGE_PREFIX    = "stor-"
	COLLECTION_PREFIX = "coll-"
	MASTER_PREFIX     = "mas-"
	CACHE_PREFIX      = "ca-"
)

type Database struct {
	db          *badger.DB
	fs          map[string]filesystem.FileSystem
	storages    map[string]*Storage
	collections map[string]*Collection
	estates     map[string]*Estate
}

func (db *Database) Init() error {
	db.collections = make(map[string]*Collection)
	db.storages = make(map[string]*Storage)
	// load storages
	if err := db.IteratePrefix(STORAGE_PREFIX, func(val []byte) error {
		stor := &Storage{}
		if err := json.Unmarshal(val, stor); err != nil {
			return emperror.Wrapf(err, "cannot unmarshal storage for %s", string(val))
		}
		stor.db = db
		fs, ok := db.fs[stor.FSType]
		if !ok {
			return fmt.Errorf("invalid filesystem type %s for %s", stor.FSType, stor.Id)
		}
		stor.fs = fs
		db.storages[stor.Id] = stor
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error reading storages")
	}
	// load collections
	if err := db.IteratePrefix(COLLECTION_PREFIX, func(val []byte) error {
		coll := &Collection{}
		if err := json.Unmarshal(val, coll); err != nil {
			return emperror.Wrapf(err, "cannot unmarshal storage for %s", string(val))
		}
		coll.db = db
		db.collections[coll.Id] = coll
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error reading collections")
	}
	// load estates
	if err := db.IteratePrefix(ESTATE_PREFIX, func(val []byte) error {
		est := &Estate{}
		if err := json.Unmarshal(val, est); err != nil {
			return emperror.Wrapf(err, "cannot unmarshal storage for %s", string(val))
		}
		est.db = db
		db.estates[est.Id] = est
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error reading estates")
	}
	return nil
}

func (db *Database) IteratePrefix(prefix string, f func(val []byte) error) error {
	if err := db.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{
			PrefetchValues: true,
			PrefetchSize:   0,
			Reverse:        false,
			AllVersions:    false,
			Prefix:         []byte(prefix),
			InternalAccess: false,
		})
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if err := item.Value(f); err != nil {
				return emperror.Wrapf(err, "cannot get value of %s", string(item.Key()))
			}
		}
		return nil
	}); err != nil {
		return emperror.Wrapf(err, "error iterating %s", prefix)
	}
	return nil
}
func (db *Database) Get(key string) ([]byte, error) {
	item, err := db.db.NewTransaction(false).Get([]byte(key))
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot load %s", key)
	}
	var result []byte
	if err := item.Value(func(val []byte) error {
		result = val
		return nil
	}); err != nil {
		return nil, emperror.Wrapf(err, "cannot load value of %s", key)
	}
	return result, nil
}

func (db *Database) GetEstate(id string) (*Estate, error) {
	est, ok := db.estates[id]
	if !ok {
		return nil, fmt.Errorf("estate %s not found", id)
	}
	return est, nil
}
func (db *Database) AddEstate(est *Estate) error {
	key := est.GetKey()
	_, err := db.db.NewTransaction(false).Get([]byte(key))
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return emperror.Wrapf(err, "error checking for key %s", key)
		}
	} else {
		return fmt.Errorf("key %s already in database", key)
	}
	est.db = db
	jsonbytes, err := json.Marshal(est)
	if err != nil {
		return emperror.Wrapf(err, "cannot marshal estate %s", est.Id)
	}
	if err := db.db.NewTransaction(true).Set([]byte(key), jsonbytes); err != nil {
		return emperror.Wrapf(err, "cannot store estate %s as %s", est.Id, key)
	}

	db.estates[est.Id] = est

	return nil
}

func (db *Database) GetStorage(id string) (*Storage, error) {
	stor, ok := db.storages[id]
	if !ok {
		return nil, fmt.Errorf("storage %s not found", id)
	}
	return stor, nil
}
func (db *Database) AddStorage(stor *Storage) error {
	key := stor.GetKey()
	_, err := db.db.NewTransaction(false).Get([]byte(key))
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return emperror.Wrapf(err, "error checking for key %s", key)
		}
	} else {
		return fmt.Errorf("key %s already in database", key)
	}
	stor.db = db
	jsonbytes, err := json.Marshal(stor)
	if err != nil {
		return emperror.Wrapf(err, "cannot marshal filesystem %s", stor.Id)
	}
	if err := db.db.NewTransaction(true).Set([]byte(key), jsonbytes); err != nil {
		return emperror.Wrapf(err, "cannot store storage %s as %s", stor.Id, key)
	}

	db.storages[stor.Id] = stor

	return nil
}

func (db *Database) GetCollection(id string) (*Collection, error) {
	coll, ok := db.collections[id]
	if !ok {
		return nil, fmt.Errorf("collection %s not found", id)
	}
	return coll, nil
}

func (db *Database) GetMaster(id string) (*Master, error) {
	key := MASTER_PREFIX + id
	val, err := db.Get(key)
	if err != nil {
		return nil, emperror.Wrap(err, "cannot get master")
	}
	master := &Master{}
	if err := json.Unmarshal(val, master); err != nil {
		return nil, emperror.Wrapf(err, "cannot unmarshal master %s - %s", key, string(val))
	}
	master.db = db
	master.coll, err = db.GetCollection(master.CollectionId)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot load collection %s for master %s", master.CollectionId, id)
	}
	return master, nil
}
