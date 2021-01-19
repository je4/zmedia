package database

import "errors"

var ErrNotFound = errors.New("database: could not find entry")

type Database interface {
	GetStorages(mdb *MediaDatabase, callback func(storage *Storage) error) error
	GetStorageById(mdb *MediaDatabase, storageid int64) (*Storage, error)
	GetStorageByName(mdb *MediaDatabase, name string) (*Storage, error)
	CreateStorage(mdb *MediaDatabase, name string, fsname, jwtKey string) (*Storage, error)

	GetEstates(mdb *MediaDatabase, callback func(estate *Estate) error) error
	GetEstateById(mdb *MediaDatabase, EstateID int64) (*Estate, error)
	GetEstateByName(mdb *MediaDatabase, Name string) (*Estate, error)
	CreateEstate(mdb *MediaDatabase, name, description string) (*Estate, error)

	GetCollections(mdb *MediaDatabase, callback func(storage *Collection) error) error
	GetCollectionById(mdb *MediaDatabase, CollectionId int64) (*Collection, error)
	GetCollectionByName(mdb *MediaDatabase, Name string) (*Collection, error)
	CreateCollection(db *MediaDatabase, name string, estate *Estate, storage *Storage, signature_prefix, description string, zoterogroup int64) (*Collection, error)

	GetMaster(mdb *MediaDatabase, collection *Collection, signature string) (*Master, error)
	GetMasterById(mdb *MediaDatabase, collection *Collection, masterid int64) (*Master, error)
	CreateMaster(mdb *MediaDatabase, collection *Collection, signature, urn string, parent *Master) (*Master, error)
	StoreMaster(db *MediaDatabase, m *Master) error

	GetCacheByMaster(mdb *MediaDatabase, master *Master, action string, paramstr string) (*Cache, error)
	GetCache(mdb *MediaDatabase, collection, signature, action string, paramstr string) (*Cache, error)
	StoreCache(mdb *MediaDatabase, cache *Cache) error
}
