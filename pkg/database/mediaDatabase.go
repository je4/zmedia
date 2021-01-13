package database

import (
	"fmt"
	"github.com/bluele/gcache"
	"github.com/goph/emperror"
	"github.com/gosimple/slug"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DataType int64

const DT_Storage DataType = 1
const DT_Collection DataType = 2
const DT_Estate DataType = 3
const DT_Master DataType = 4
const DT_Cache DataType = 5

type MediaDatabase struct {
	mutex             map[DataType]*sync.Mutex
	db                Database
	fss               map[string]filesystem.FileSystem
	storages          map[int64]*Storage
	collectionsById   map[int64]*Collection
	collectionsByName map[string]*Collection
	estates           map[int64]*Estate
	cache             gcache.Cache
}

func NewMediaDatabase(db Database, fss map[string]filesystem.FileSystem) (*MediaDatabase, error) {
	mdb := &MediaDatabase{
		db:    db,
		fss:   fss,
		cache: gcache.New(50).ARC().Expiration(3 * time.Hour).Build(),
		mutex: map[DataType]*sync.Mutex{
			DT_Storage:    {},
			DT_Collection: {},
			DT_Estate:     {},
			DT_Master:     {},
			DT_Cache:      {},
		},
	}
	mdb.Init()

	return mdb, nil
}

func (db *MediaDatabase) Init() error {
	db.estates = make(map[int64]*Estate)
	db.db.GetEstates(db, func(est *Estate) error {
		db.estates[est.Id] = est
		return nil
	})

	db.storages = make(map[int64]*Storage)
	db.db.GetStorages(db, func(stor *Storage) error {
		db.storages[stor.Id] = stor
		return nil
	})

	db.collectionsById = make(map[int64]*Collection)
	db.db.GetCollections(db, func(coll *Collection) error {
		db.collectionsById[coll.Id] = coll
		db.collectionsByName[strings.ToLower(coll.Name)] = coll
		return nil
	})
	return nil
}

func (db *MediaDatabase) GetEstateById(id int64) (*Estate, error) {
	db.mutex[DT_Estate].Lock()
	defer db.mutex[DT_Estate].Unlock()
	key := "est-" + strconv.FormatInt(id, 10)
	cval, err := db.cache.Get(key)
	var est *Estate
	var ok bool
	if err == nil {
		est, ok = cval.(*Estate)
		if ok {
			return est, nil
		}
	}
	est, err = db.db.GetEstateById(db, id)
	if err == nil {
		db.cache.Set(key, est)
		db.cache.Set("est-"+est.Name, est)
	}
	return est, err
}
func (db *MediaDatabase) GetEstateByName(name string) (*Estate, error) {
	name = strings.ToLower(name)
	db.mutex[DT_Estate].Lock()
	defer db.mutex[DT_Estate].Unlock()
	key := "est-" + name
	cval, err := db.cache.Get(key)
	var est *Estate
	var ok bool
	if err == nil {
		est, ok = cval.(*Estate)
		if ok {
			return est, nil
		}
	}
	est, err = db.db.GetEstateByName(db, name)
	if err == nil {
		db.cache.Set(key, est)
		db.cache.Set("est-"+strconv.FormatInt(est.Id, 10), est)
	}
	return est, err
}
func (db *MediaDatabase) CreateEstate(name, description string) (*Estate, error) {
	name = strings.ToLower(name)
	return db.db.CreateEstate(db, name, description)
}

func (db *MediaDatabase) GetStorageById(id int64) (*Storage, error) {
	db.mutex[DT_Storage].Lock()
	defer db.mutex[DT_Storage].Unlock()
	key := "stor-" + strconv.FormatInt(id, 10)
	cval, err := db.cache.Get(key)
	var stor *Storage
	var ok bool
	if err == nil {
		stor, ok = cval.(*Storage)
		if ok {
			return stor, nil
		}
	}
	stor, err = db.db.GetStorageById(db, id)
	if err == nil {
		db.cache.Set(key, stor)
		db.cache.Set("stor-"+stor.Name, stor)
	}
	return stor, err
}
func (db *MediaDatabase) GetStorageByName(name string) (*Storage, error) {
	name = strings.ToLower(name)
	db.mutex[DT_Storage].Lock()
	defer db.mutex[DT_Storage].Unlock()
	key := "stor-" + name
	cval, err := db.cache.Get(key)
	var stor *Storage
	var ok bool
	if err == nil {
		stor, ok = cval.(*Storage)
		if ok {
			return stor, nil
		}
	}
	stor, err = db.db.GetStorageByName(db, name)
	if err == nil {
		db.cache.Set(key, stor)
		db.cache.Set("stor-"+strconv.FormatInt(stor.Id, 10), stor)
	}
	return stor, err
}
func (db *MediaDatabase) CreateStorage(name string, fsname, jwtKey string) (*Storage, error) {
	name = strings.ToLower(name)
	fname := "media-" + slug.Make(name)
	fs, ok := db.fss[fsname]
	if !ok {
		return nil, fmt.Errorf("filesystem %s unknown", fsname)
	}

	exists, err := fs.BucketExists(fname)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot check folder %s/%s", fs.String(), fname)
	}
	if exists {
		return nil, fmt.Errorf("folder %s/%s already exists", fs.String(), fname)
	}
	if err := fs.BucketCreate(fname, filesystem.FolderCreateOptions{}); err != nil {
		return nil, emperror.Wrapf(err, "cannot create folder %s/%s", fs.String(), fname)
	}

	/*
		// create subfolders with cache directories
		for _, sub := range []string{"data", "video", "submaster", "temp"} {
			dirname := fname + "/" + sub
			if err := fs.BucketCreate(dirname, filesystem.FolderCreateOptions{}); err != nil {
				return nil, emperror.Wrapf(err, "cannot create folder %s/%s", fs.String(), dirname)
			}
			for _, hex := range []rune("0123456789abcdef") {
				hname := dirname + "/" + string([]rune{hex})
				if err := fs.BucketCreate(hname, filesystem.FolderCreateOptions{}); err != nil {
					return nil, emperror.Wrapf(err, "cannot create folder %s/%s", fs.String(), hname)
				}
				for _, hex2 := range []rune("0123456789abcdef") {
					hname2 := hname + "/" + string([]rune{hex2})
					if err := fs.BucketCreate(hname2, filesystem.FolderCreateOptions{}); err != nil {
						return nil, emperror.Wrapf(err, "cannot create folder %s/%s", fs.String(), hname2)
					}
				}
			}
		}
	*/
	return db.db.CreateStorage(db, name, fsname, jwtKey)
}

func (db *MediaDatabase) GetCollectionById(id int64) (*Collection, error) {
	db.mutex[DT_Collection].Lock()
	defer db.mutex[DT_Collection].Unlock()
	key := "coll-" + strconv.FormatInt(id, 10)
	cval, err := db.cache.Get(key)
	var coll *Collection
	var ok bool
	if err == nil {
		coll, ok = cval.(*Collection)
		if ok {
			return coll, nil
		}
	}
	coll, err = db.db.GetCollectionById(db, id)
	if err == nil {
		db.cache.Set(key, coll)
		db.cache.Set("coll-"+coll.Name, coll)
	}
	return coll, err
}
func (db *MediaDatabase) GetCollectionByName(name string) (*Collection, error) {
	name = strings.ToLower(name)
	db.mutex[DT_Collection].Lock()
	defer db.mutex[DT_Collection].Unlock()
	key := "coll-" + name
	cval, err := db.cache.Get(key)
	var coll *Collection
	var ok bool
	if err == nil {
		coll, ok = cval.(*Collection)
		if ok {
			return coll, nil
		}
	}
	coll, err = db.db.GetCollectionByName(db, name)
	if err == nil {
		db.cache.Set(key, coll)
		db.cache.Set("coll-"+strconv.FormatInt(coll.Id, 10), coll)
	}
	return coll, err
}
func (db *MediaDatabase) CreateCollection(name string, estate *Estate, storage *Storage, signature_prefix, description string, zoterogroup int64) (*Collection, error) {
	name = strings.ToLower(name)
	return db.db.CreateCollection(db, name, estate, storage, signature_prefix, description, zoterogroup)
}

func (db *MediaDatabase) GetMaster(collection *Collection, signature string) (*Master, error) {
	signature = strings.ToLower(signature)
	var master *Master
	var ok bool
	key := fmt.Sprintf("mas-%s/%s", collection.Name, signature)
	cval, err := db.cache.Get(key)
	if err == nil {
		master, ok = cval.(*Master)
		if ok {
			return master, nil
		}
	}
	master, err = db.db.GetMaster(db, collection, signature)
	if err != nil {
		return nil, err
	}
	if err := db.cache.Set(key, master); err != nil {
		return nil, emperror.Wrapf(err, "cannot store master %s in cache", key)
	}
	return master, nil
}
func (db *MediaDatabase) GetMasterById(collection *Collection, masterid int64) (*Master, error) {
	db.mutex[DT_Master].Lock()
	defer db.mutex[DT_Master].Unlock()
	var master *Master
	var ok bool
	key := fmt.Sprintf("mas-%s/%d", collection.Name, masterid)
	cval, err := db.cache.Get(key)
	if err == nil {
		master, ok = cval.(*Master)
		if ok {
			return master, nil
		}
	}
	master, err = db.db.GetMasterById(db, collection, masterid)
	if err != nil {
		return nil, err
	}
	if err := db.cache.Set(key, master); err != nil {
		return nil, emperror.Wrapf(err, "cannot store master %s in cache", key)
	}
	return master, nil
}

func (db *MediaDatabase) GetCache(collection, signature, action string, paramstr string) (*Cache, error) {
	action = strings.ToLower(action)
	lparams := []string{}
	for _, param := range strings.Split(paramstr, "/") {
		lparams = append(lparams, strings.ToLower(param))
	}
	sort.Strings(lparams)
	paramstr = strings.Join(lparams, "/")

	return db.db.GetCache(db, collection, signature, action, paramstr)
}
func (db *MediaDatabase) GetCacheByMaster(master *Master, action string, paramstr string) (*Cache, error) {
	action = strings.ToLower(action)
	lparams := []string{}
	for _, param := range strings.Split(paramstr, "/") {
		lparams = append(lparams, strings.ToLower(param))
	}
	sort.Strings(lparams)
	paramstr = strings.Join(lparams, "/")

	return db.db.GetCacheByMaster(db, master, action, paramstr)
}
