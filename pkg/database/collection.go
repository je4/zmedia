package database

import "fmt"

type Collection struct {
	db          *MediaDatabase `json:"-"`
	Id          int64          `json:"id"`
	EstateId    int64          `json:"estateid"`
	StorageId   int64          `json:"storageid"`
	Estate      *Estate        `json:"-"`
	Storage     *Storage       `json:"-"`
	Name        string         `json:"string"`
	Description string         `json:"description,omitempty"`
	ZoteroGroup int64          `json:"zoterogroup,omitempty"`
}

func NewCollection(mdb *MediaDatabase, id int64, storageid, estateid int64, name, description string, zoteroGroup int64) (*Collection, error) {
	storage, ok := mdb.storages[storageid]
	if !ok {
		return nil, fmt.Errorf("cannot find storage with id %v for collection [%v] %s", storageid, id, name)
	}
	estate, ok := mdb.estates[estateid]
	if !ok {
		return nil, fmt.Errorf("cannot find estate with id %v for collection [%v] %s", estateid, id, name)
	}
	coll := &Collection{
		db:          mdb,
		Id:          id,
		Storage:     storage,
		Estate:      estate,
		StorageId:   storage.Id,
		EstateId:    estate.Id,
		Name:        name,
		Description: description,
		ZoteroGroup: zoteroGroup,
	}
	return coll, nil
}
