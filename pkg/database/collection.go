package database

import (
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/goph/emperror"
)

type Collection struct {
	db          *Database `json:"-"`
	Id          string    `json:"id"`
	EstateId    string    `json:"estateid"`
	StorageId   string    `json:"storageid"`
	Name        string    `json:"string"`
	Description string    `json:"description,omitempty"`
	ZoteroGroup int64     `json:"zoterogroup,omitempty"`
}

func NewCollection(id, estateId, name, description string, zoteroGroup int64) *Collection {
	coll := &Collection{
		Id:          id,
		EstateId:    estateId,
		Name:        name,
		Description: description,
		ZoteroGroup: zoteroGroup,
	}
	return coll
}

func (coll *Collection) GetKey() string {
	return COLLECTION_PREFIX + coll.Id
}

func (coll *Collection) AddMaster(mas *Master) (*Master, error) {
	key := mas.GetKey()
	_, err := mas.db.db.NewTransaction(false).Get([]byte(key))
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return nil, emperror.Wrapf(err, "error checking for key %s", key)
		}
	} else {
		return nil, fmt.Errorf("key %s already in database", key)
	}
	mas.db = mas.db
	mas.CollectionId = coll.Id
	jsonbytes, err := json.Marshal(mas)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot marshal master %s", mas.Id)
	}
	if err := mas.db.db.NewTransaction(true).Set([]byte(key), jsonbytes); err != nil {
		return nil, emperror.Wrapf(err, "cannot store master %s as %s", mas.Id, key)
	}
	return mas, nil
}
