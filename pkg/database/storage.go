package database

import (
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/goph/emperror"
	"github.com/je4/zmedia/v2/pkg/filesystem"
)

type Storage struct {
	fs       filesystem.FileSystem `json:"-"`
	db       *Database             `json:"-"`
	FSType   string                `json:"fstype"`
	Id       string                `json:"id"`
	Name     string                `json:"name"`
	Filebase string                `json:"filebase"`
	DataDir  string                `json:"data"`
	VideoDir string                `json:"video"`
	TempDir  string                `json:"temp"`
}

func NewStorage(fs filesystem.FileSystem, id, name, filebase, datadir, videodir, tempdir string) (*Storage, error) {
	if datadir == "" {
		datadir = "data"
	}
	if videodir == "" {
		videodir = "video"
	}
	if tempdir == "" {
		tempdir = "temp"
	}

	if err := fs.FolderCreate(filebase, filesystem.FolderCreateOptions{}); err != nil {
		return nil, emperror.Wrapf(err, "cannot create folder %s", filebase)
	}

	stor := &Storage{
		fs:       fs,
		Id:       id,
		FSType:   fs.String(),
		Name:     name,
		Filebase: filebase,
		DataDir:  datadir,
		VideoDir: videodir,
		TempDir:  tempdir,
	}
	return stor, nil
}

func (stor *Storage) GetKey() string {
	return STORAGE_PREFIX + stor.Id
}

func (stor *Storage) AddCollection(coll *Collection) error {
	key := coll.GetKey()
	_, err := stor.db.db.NewTransaction(false).Get([]byte(key))
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return emperror.Wrapf(err, "error checking for key %s", key)
		}
	} else {
		return fmt.Errorf("key %s already in database", key)
	}
	coll.db = stor.db
	coll.StorageId = stor.Id
	jsonbytes, err := json.Marshal(coll)
	if err != nil {
		return emperror.Wrapf(err, "cannot marshal collection %s", coll.Id)
	}
	if err := stor.db.db.NewTransaction(true).Set([]byte(key), jsonbytes); err != nil {
		return emperror.Wrapf(err, "cannot store collection %s as %s", coll.Id, key)
	}
	stor.db.collections[coll.Id] = coll
	return nil
}
