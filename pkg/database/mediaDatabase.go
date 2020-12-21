package database

import (
	"fmt"
	"github.com/bluele/gcache"
	"github.com/goph/emperror"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"strings"
	"time"
)

type MediaDatabase struct {
	db                Database
	fss               map[string]filesystem.FileSystem
	storages          map[int64]*Storage
	collectionsById   map[int64]*Collection
	collectionsByName map[string]*Collection
	estates           map[int64]*Estate
	cache             gcache.Cache
}

func NewMediaDYatabase(db Database, fss map[string]filesystem.FileSystem) (*MediaDatabase, error) {
	mdb := &MediaDatabase{
		db:    db,
		fss:   fss,
		cache: gcache.New(50).ARC().Expiration(3 * time.Hour).Build(),
	}
	mdb.Init()

	return mdb, nil
}

func (db *MediaDatabase) Init() error {
	db.estates = make(map[int64]*Estate)
	db.db.LoadEstates(db, func(est *Estate) error {
		db.estates[est.Id] = est
		return nil
	})

	db.storages = make(map[int64]*Storage)
	db.db.LoadStorages(func(stor *Storage) error {
		db.storages[stor.Id] = stor
		return nil
	})

	db.collectionsById = make(map[int64]*Collection)
	db.db.LoadCollections(db, func(coll *Collection) error {
		db.collectionsById[coll.Id] = coll
		db.collectionsByName[strings.ToLower(coll.Name)] = coll
		return nil
	})
	return nil
}

func (db *MediaDatabase) GetEstate(id int64) (*Estate, error) {
	est, ok := db.estates[id]
	if !ok {
		return nil, fmt.Errorf("estate %v not found", id)
	}
	return est, nil
}

func (db *MediaDatabase) GetStorage(id int64) (*Storage, error) {
	stor, ok := db.storages[id]
	if !ok {
		return nil, fmt.Errorf("storage %v not found", id)
	}
	return stor, nil
}

func (db *MediaDatabase) GetCollectionById(id int64) (*Collection, error) {
	coll, ok := db.collectionsById[id]
	if !ok {
		return nil, fmt.Errorf("collection %s not found", id)
	}
	return coll, nil
}
func (db *MediaDatabase) GetCollectionByName(name string) (*Collection, error) {
	coll, ok := db.collectionsByName[name]
	if !ok {
		return nil, fmt.Errorf("collection %s not found", name)
	}
	return coll, nil
}

func (db *MediaDatabase) GetMaster(collection *Collection, signature string) (*Master, error) {
	var master *Master
	var ok bool
	key := fmt.Sprintf("%s/%s", collection.Name, signature)
	cval, err := db.cache.Get(key)
	if err == nil {
		master, ok = cval.(*Master)
		if ok {
			return master, nil
		}
	}
	master, err = db.db.GetMaster(collection, signature)
	if err != nil {
		return nil, err
	}
	if err := db.cache.Set(key, master); err != nil {
		return nil, emperror.Wrapf(err, "cannot store master %s in cache", key)
	}
	return master, nil
}
