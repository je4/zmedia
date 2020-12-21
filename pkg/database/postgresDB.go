package database

import (
	"database/sql"
	"fmt"
	"github.com/goph/emperror"
	"github.com/op/go-logging"
	"sync"
)

type DataType int64

const DT_Storage DataType = 1
const DT_Collection DataType = 2
const DT_Estate DataType = 3
const DT_Master DataType = 4
const DT_Cache DataType = 5

type PostgresDB struct {
	mutex  map[DataType]*sync.Mutex
	db     *sql.DB
	schema string
	logger *logging.Logger
}

func NewPostgresDB(db *sql.DB, schema string, logger *logging.Logger) (*PostgresDB, error) {
	pgdb := &PostgresDB{
		db:     db,
		schema: schema,
		logger: logger,
		mutex: map[DataType]*sync.Mutex{
			DT_Storage:    {},
			DT_Collection: {},
			DT_Estate:     {},
			DT_Master:     {},
			DT_Cache:      {},
		},
	}
	return pgdb, nil
}

func (db *PostgresDB) LoadStorages(mdb *MediaDatabase, callback func(storage *Storage) error) error {
	db.mutex[DT_Storage].Lock()
	defer db.mutex[DT_Storage].Unlock()
	var sqlstr string = fmt.Sprintf("SELECT storageid, name, urlbase, filebase, datadir, videodir, submasterdir, tempdir, jwtkey FROM %s.storage", db.schema)
	db.logger.Debugf("SQL: %s", sqlstr)
	rows, err := db.db.Query(sqlstr)
	if err != nil {
		return emperror.Wrapf(err, "cannot execute sql %s", sqlstr)
	}
	defer rows.Close()
	var StorageID int64
	var Name string
	var URLBase, FileBase string
	var DataDir, VideoDir, SubmasterDir, TempDir string
	var JWTKey string
	for rows.Next() {
		if err := rows.Scan(&StorageID, &Name, &URLBase, &FileBase, &DataDir, &VideoDir, &SubmasterDir, &TempDir, &JWTKey); err != nil {
			return emperror.Wrapf(err, "cannot scan result from %s", sqlstr)
		}
		stor, err := NewStorage(mdb, StorageID, Name, FileBase, DataDir, VideoDir, SubmasterDir, TempDir, JWTKey)
		if err != nil {
			return emperror.Wrapf(err, "cannot instantiate storage [%v] %s", StorageID, Name)
		}
		if err := callback(stor); err != nil {
			return emperror.Wrapf(err, "cannot callback for storage [%v] %s", StorageID, Name)
		}
	}
	return nil
}

func (db *PostgresDB) LoadEstates(mdb *MediaDatabase, callback func(estate *Estate) error) error {
	db.mutex[DT_Estate].Lock()
	defer db.mutex[DT_Estate].Unlock()
	var sqlstr string = fmt.Sprintf("SELECT estateid, name, description FROM %s.estate", db.schema)
	db.logger.Debugf("SQL: %s", sqlstr)
	rows, err := db.db.Query(sqlstr)
	if err != nil {
		return emperror.Wrapf(err, "cannot execute sql %s", sqlstr)
	}
	defer rows.Close()
	var EstateID int64
	var Name, Description string
	for rows.Next() {
		if err := rows.Scan(&EstateID, &Name, &Description); err != nil {
			return emperror.Wrapf(err, "cannot scan result from %s", sqlstr)
		}
		est, err := NewEstate(mdb, EstateID, Name, Description)
		if err != nil {
			return emperror.Wrapf(err, "cannot instantiate estate [%v] %s", EstateID, Name)
		}
		if err := callback(est); err != nil {
			return emperror.Wrapf(err, "cannot callback for estate [%v] %s", EstateID, Name)
		}
	}
	return nil
}

func (db *PostgresDB) LoadCollections(mdb *MediaDatabase, callback func(storage *Collection) error) error {
	db.mutex[DT_Collection].Lock()
	defer db.mutex[DT_Collection].Unlock()
	var sqlstr string = fmt.Sprintf("SELECT collectionid, estateid, storageid , name, description, signature_prefix, json , zoterogroup FROM %s.storage", db.schema)
	db.logger.Debugf("SQL: %s", sqlstr)
	rows, err := db.db.Query(sqlstr)
	if err != nil {
		return emperror.Wrapf(err, "cannot execute sql %s", sqlstr)
	}
	defer rows.Close()
	var CollectionId int64
	var StorageID int64
	var EstateID int64
	var Name string
	var Description, SignaturePrefix, JSONStr string
	var ZoteroGroup int64
	for rows.Next() {
		if err := rows.Scan(&CollectionId, &EstateID, &StorageID, &Name, &Description, &SignaturePrefix, &JSONStr, &ZoteroGroup); err != nil {
			return emperror.Wrapf(err, "cannot scan result from %s", sqlstr)
		}
		coll, err := NewCollection(mdb, CollectionId, StorageID, EstateID, Name, Description, ZoteroGroup)
		if err != nil {
			return emperror.Wrapf(err, "cannot instantiate collection [%v] %s", CollectionId, Name)
		}
		if err := callback(coll); err != nil {
			return emperror.Wrapf(err, "cannot callback for storage [%v] %s", StorageID, Name)
		}
	}
	return nil
}

func (db *PostgresDB) GetMaster(mdb *MediaDatabase, collection *Collection, signature string) (*Master, error) {
	sqlstr := fmt.Sprintf("SELECT masterid, urn, type, subtype, objecttype, status, parentid, mimetype, error FROM %s.master WHERE collectinid=$1, signature=$2", db.schema)
	row := db.db.QueryRow(sqlstr, collection.Id, signature)
	var MasterId int64
	var URN, Type, SubType, ObjectType, Status string
	var ParentId int64
	var Mimetype, ErrStatus string
	switch err := row.Scan(&MasterId, &URN, &Type, &SubType, &ObjectType, &Status, &ParentId, &Mimetype, &ErrStatus); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("master %s/%s does not exist", collection.Name, signature)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load master %s/%s", collection.Name, signature)
	}
	master, err := NewMaster(mdb, MasterId, URN, Type, SubType, Mimetype, ObjectType, "", nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate master %s/%s", collection.Name, signature)
	}
	return master, nil
}
