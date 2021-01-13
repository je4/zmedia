package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	"github.com/gosimple/slug"
	"github.com/op/go-logging"
	"strings"
)

type PostgresDB struct {
	db     *sql.DB
	schema string
	logger *logging.Logger
}

func NewPostgresDB(db *sql.DB, schema string, logger *logging.Logger) (*PostgresDB, error) {
	pgdb := &PostgresDB{
		db:     db,
		schema: schema,
		logger: logger,
	}
	return pgdb, nil
}

func (db *PostgresDB) GetStorages(mdb *MediaDatabase, callback func(storage *Storage) error) error {
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
func (db *PostgresDB) GetStorageById(mdb *MediaDatabase, storageid int64) (*Storage, error) {
	var sqlstr string = fmt.Sprintf("SELECT name, urlbase, filebase, datadir, videodir, submasterdir, tempdir, jwtkey FROM %s.storage WHERE storageid=$1", db.schema)
	params := []interface{}{storageid}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var Name string
	var URLBase, FileBase string
	var DataDir, VideoDir, SubmasterDir, TempDir string
	var JWTKey sql.NullString
	switch err := row.Scan(&Name, &URLBase, &FileBase, &DataDir, &VideoDir, &SubmasterDir, &TempDir, &JWTKey); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("storage #%v does not exist", storageid)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load storage #$v", storageid)
	}
	stor, err := NewStorage(mdb, storageid, Name, FileBase, DataDir, VideoDir, SubmasterDir, TempDir, JWTKey.String)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate storage [%v] %s", storageid, Name)
	}

	return stor, nil
}
func (db *PostgresDB) GetStorageByName(mdb *MediaDatabase, name string) (*Storage, error) {
	name = strings.ToLower(name)
	var sqlstr string = fmt.Sprintf("SELECT storageid, urlbase, filebase, datadir, videodir, submasterdir, tempdir, jwtkey FROM %s.storage WHERE name=$1", db.schema)
	params := []interface{}{name}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var StorageId int64
	var URLBase, FileBase string
	var DataDir, VideoDir, SubmasterDir, TempDir string
	var JWTKey sql.NullString
	switch err := row.Scan(&StorageId, &URLBase, &FileBase, &DataDir, &VideoDir, &SubmasterDir, &TempDir, &JWTKey); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("storage %v does not exist", name)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load storage $v", name)
	}
	stor, err := NewStorage(mdb, StorageId, name, FileBase, DataDir, VideoDir, SubmasterDir, TempDir, JWTKey.String)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate storage [%v] %s", StorageId, name)
	}

	return stor, nil
}
func (db *PostgresDB) CreateStorage(mdb *MediaDatabase, name string, fsname, jwtKey string) (*Storage, error) {
	fname := "media-" + slug.Make(name)
	sqlstr := fmt.Sprintf("INSERT INTO %s.storage (name, urlbase, filebase, datadir, videodir, submasterdir, tempdir, jwtkey) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING storageid", db.schema)
	params := []interface{}{
		name,
		"",
		fmt.Sprintf("fs://%s/%s", fsname, fname),
		"data",
		"video",
		"submaster",
		"temp",
		jwtKey,
	}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var LastInsertId int64
	if err := row.Scan(&LastInsertId); err != nil {
		return nil, emperror.Wrapf(err, "cannot create database entry for %s - %s %v", name, sqlstr, params)
	}
	return mdb.GetStorageById(LastInsertId)
}

func (db *PostgresDB) GetEstates(mdb *MediaDatabase, callback func(estate *Estate) error) error {
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
func (db *PostgresDB) GetEstateById(mdb *MediaDatabase, EstateID int64) (*Estate, error) {
	var sqlstr string = fmt.Sprintf("SELECT name, description FROM %s.estate WHERE estateid=$1", db.schema)
	params := []interface{}{EstateID}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var Name, Description string
	switch err := row.Scan(&Name, &Description); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("estate #%v does not exist", EstateID)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load estate #$v", EstateID)
	}
	est, err := NewEstate(mdb, EstateID, Name, Description)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate estate [%v] %s", EstateID, Name)
	}
	return est, nil
}
func (db *PostgresDB) GetEstateByName(mdb *MediaDatabase, Name string) (*Estate, error) {
	Name = strings.ToLower(Name)
	var sqlstr string = fmt.Sprintf("SELECT estateid, description FROM %s.estate WHERE name=$1", db.schema)
	params := []interface{}{Name}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var EstateID int64
	var Description string
	switch err := row.Scan(&EstateID, &Description); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("estate %v does not exist", Name)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load storage $v", Name)
	}
	est, err := NewEstate(mdb, EstateID, Name, Description)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate estate [%v] %s", EstateID, Name)
	}
	return est, nil
}
func (db *PostgresDB) CreateEstate(mdb *MediaDatabase, name, description string) (*Estate, error) {
	sqlstr := fmt.Sprintf("INSERT INTO %s.estate (name, description) VALUES($1, $2) RETURNING estateid", db.schema)
	params := []interface{}{
		name,
		description,
	}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var LastInsertId int64
	if err := row.Scan(&LastInsertId); err != nil {
		return nil, emperror.Wrapf(err, "cannot create database entry for %s - %s %v", name, sqlstr, params)
	}
	return mdb.GetEstateById(LastInsertId)

}

func (db *PostgresDB) GetCollections(mdb *MediaDatabase, callback func(storage *Collection) error) error {
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
		storage, err := mdb.GetStorageById(StorageID)
		if err != nil {
			return emperror.Wrapf(err, "cannot get storage #%v", StorageID)
		}
		estate, err := mdb.GetEstateById(EstateID)
		if err != nil {
			return emperror.Wrapf(err, "cannot get estate #%v", EstateID)
		}
		coll, err := NewCollection(mdb, CollectionId, storage, estate, Name, Description, ZoteroGroup)
		if err != nil {
			return emperror.Wrapf(err, "cannot instantiate collection [%v] %s", CollectionId, Name)
		}
		if err := callback(coll); err != nil {
			return emperror.Wrapf(err, "cannot callback for storage [%v] %s", StorageID, Name)
		}
	}
	return nil
}
func (db *PostgresDB) GetCollectionById(mdb *MediaDatabase, CollectionId int64) (*Collection, error) {
	var sqlstr string = fmt.Sprintf("SELECT estateid, storageid , name, description, signature_prefix, json , zoterogroup FROM %s.storage WHERE collectionid=$1", db.schema)
	params := []interface{}{CollectionId}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var StorageID int64
	var EstateID int64
	var Name string
	var Description, SignaturePrefix, JSONStr string
	var ZoteroGroup int64
	switch err := row.Scan(&EstateID, &StorageID, &Name, &Description, &SignaturePrefix, &JSONStr, &ZoteroGroup); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("collection #%v does not exist", CollectionId)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load collection #$v", CollectionId)
	}
	storage, err := mdb.GetStorageById(StorageID)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get storage #%v", StorageID)
	}
	estate, err := mdb.GetEstateById(EstateID)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get estate #%v", EstateID)
	}
	coll, err := NewCollection(mdb, CollectionId, storage, estate, Name, Description, ZoteroGroup)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate collection [%v] %s", CollectionId, Name)
	}
	return coll, nil
}
func (db *PostgresDB) GetCollectionByName(mdb *MediaDatabase, Name string) (*Collection, error) {
	Name = strings.ToLower(Name)
	var sqlstr string = fmt.Sprintf("SELECT collectionid, estateid, storageid, description, signature_prefix, json , zoterogroup FROM %s.collection WHERE name=$1", db.schema)
	params := []interface{}{Name}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var CollectionId int64
	var StorageID int64
	var EstateID int64
	var Description, SignaturePrefix string
	var JSONStr sql.NullString
	var ZoteroGroup int64
	switch err := row.Scan(&CollectionId, &EstateID, &StorageID, &Description, &SignaturePrefix, &JSONStr, &ZoteroGroup); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("collection %v does not exist", Name)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load collection $v", Name)
	}
	storage, err := mdb.GetStorageById(StorageID)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get storage #%v", StorageID)
	}
	estate, err := mdb.GetEstateById(EstateID)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get estate #%v", EstateID)
	}
	coll, err := NewCollection(mdb, CollectionId, storage, estate, Name, Description, ZoteroGroup)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate collection [%v] %s", CollectionId, Name)
	}
	return coll, nil
}
func (db *PostgresDB) CreateCollection(mdb *MediaDatabase, name string, estate *Estate, storage *Storage, signaturePrefix, description string, zoteroGroup int64) (*Collection, error) {
	sqlstr := fmt.Sprintf("INSERT INTO %s.collection (estateid, name, description, signature_prefix, storageid, zoterogroup) VALUES($1, $2, $3, $4, $5, $6) RETURNING collectionid", db.schema)
	params := []interface{}{
		estate.Id,
		name,
		description,
		signaturePrefix,
		storage.Id,
		zoteroGroup,
	}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var LastInsertId int64
	if err := row.Scan(&LastInsertId); err != nil {
		return nil, emperror.Wrapf(err, "cannot create database entry for %s - %s %v", name, sqlstr, params)
	}
	return mdb.GetCollectionById(LastInsertId)
}

func (db *PostgresDB) GetMaster(mdb *MediaDatabase, collection *Collection, signature string) (*Master, error) {
	signature = strings.ToLower(signature)
	sqlstr := fmt.Sprintf("SELECT masterid, urn, type, subtype, objecttype, status, parentid, mimetype, error, sha256, metadata FROM %s.master WHERE collectinid=$1, signature=$2", db.schema)
	row := db.db.QueryRow(sqlstr, collection.Id, signature)
	var MasterId int64
	var URN, Type, SubType, ObjectType, Status string
	var ParentId int64
	var Mimetype, ErrStatus, SHA256, MetadataJSON string
	switch err := row.Scan(&MasterId, &URN, &Type, &SubType, &ObjectType, &Status, &ParentId, &Mimetype, &ErrStatus, &SHA256, &MetadataJSON); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("master %s/%s does not exist", collection.Name, signature)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load master %s/%s", collection.Name, signature)
	}
	var Metadata map[string]interface{}
	if err := json.Unmarshal([]byte(MetadataJSON), &Metadata); err != nil {
		return nil, emperror.Wrapf(err, "cannot unmarshal metadata for %s/%s - %s", collection.Name, signature, MetadataJSON)
	}
	master, err := NewMaster(mdb, collection, MasterId, ParentId, signature, URN, Type, SubType, Mimetype, ObjectType, SHA256, Metadata)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate master %s/%s", collection.Name, signature)
	}
	return master, nil
}
func (db *PostgresDB) GetMasterById(mdb *MediaDatabase, collection *Collection, masterid int64) (*Master, error) {
	sqlstr := fmt.Sprintf("SELECT signature, urn, type, subtype, objecttype, status, parentid, mimetype, error, sha256, metadata FROM %s.master WHERE collectinid=$1, masterid=$2", db.schema)
	params := []interface{}{collection.Id, masterid}
	db.logger.Debugf("SQL: %s - %v", sqlstr, params)
	row := db.db.QueryRow(sqlstr, params...)
	var Signature string
	var URN, Type, SubType, ObjectType, Status string
	var ParentId int64
	var Mimetype, ErrStatus, SHA256, MetadataJSON string
	switch err := row.Scan(&Signature, &URN, &Type, &SubType, &ObjectType, &Status, &ParentId, &Mimetype, &ErrStatus, &SHA256, &MetadataJSON); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("master #%d in %s does not exist", masterid, collection.Name)
	case nil:

	default:
		return nil, emperror.Wrapf(err, "cannot load master #%d from %s", masterid, collection.Name)
	}
	var Metadata map[string]interface{}
	if err := json.Unmarshal([]byte(MetadataJSON), &Metadata); err != nil {
		return nil, emperror.Wrapf(err, "cannot unmarshal metadata for %s/%s - %s", collection.Name, Signature, MetadataJSON)
	}
	master, err := NewMaster(mdb, collection, masterid, ParentId, Signature, URN, Type, SubType, Mimetype, ObjectType, SHA256, Metadata)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate master %s/%s", collection.Name, Signature)
	}
	return master, nil
}

func (db *PostgresDB) GetCache(mdb *MediaDatabase, collection, signature, action string, paramstr string) (*Cache, error) {

	sqlstr := fmt.Sprintf("SELECT m.masterid, c.cacheid, c.storageid, coll.collectionid, c.width, c.height, duration, c.mimetype, c.filesize, c.path "+
		"  FROM %s.cache AS c, %s.master AS m, %s.collection AS coll"+
		"  WHERE  coll.name=$1"+
		"     AND coll.collectionid=m.collectionid"+
		"     AND m.signature=$2"+
		"     AND m.masterid=c.masterid"+
		"     AND action=$3"+
		"     AND param=$4", db.schema, db.schema, db.schema)
	sqlparams := []interface{}{
		collection,
		signature,
		action,
		paramstr,
	}
	db.logger.Debugf("SQL: %s - %v", sqlstr, sqlparams)
	row := db.db.QueryRow(sqlstr, sqlparams...)
	var Masterid int64
	var CacheId, StorageId, CollectionId int64
	var Width, Height, Duration int64
	var Mimetype string
	var Filesize int64
	var Path string
	switch err := row.Scan(&Masterid, &CacheId, &StorageId, &CollectionId, &Width, &Height, &Duration, &Mimetype, &Filesize, &Path); err {
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:

	default:
		return nil, fmt.Errorf("cannot get cache %s/%s/%s/%s", collection, signature, action, paramstr)
	}
	cache, err := NewCache(mdb, CacheId, CollectionId, Masterid, action, paramstr, Mimetype, Filesize, Path, Width, Height, Duration)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate cache %s/%s/%s/%s", collection, signature, action, paramstr)
	}
	return cache, nil
}
func (db *PostgresDB) GetCacheByMaster(mdb *MediaDatabase, master *Master, action string, paramstr string) (*Cache, error) {

	coll, err := master.GetCollection()
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get collection from master #%v.%v", master.CollectionId, master.Id)
	}

	sqlstr := fmt.Sprintf("SELECT cacheid, storageid, collectionid, width, height, duration, mimetype, filesize, path FROM %s.cache WHERE masterid=$1, action=$2, param=$3", db.schema)
	sqlparams := []interface{}{
		master.Id,
		action,
		paramstr,
	}
	db.logger.Debugf("SQL: %s - %v", sqlstr, sqlparams)
	row := db.db.QueryRow(sqlstr, sqlparams...)
	var CacheId, StorageId, CollectionId int64
	var Width, Height, Duration int64
	var Mimetype string
	var Filesize int64
	var Path string
	switch err := row.Scan(&CacheId, &StorageId, &CollectionId, &Width, &Height, &Duration, &Mimetype, &Filesize, &Path); err {
	case sql.ErrNoRows:
		return nil, fmt.Errorf("cache %s/%s/%s/%s does not exist", coll.Name, master.Signature, action, paramstr)
	case nil:

	default:
		return nil, fmt.Errorf("cannot get cache %s/%s/%s/%s", coll.Name, master.Signature, action, paramstr)
	}
	cache, err := NewCache(mdb, CacheId, CollectionId, master.Id, action, paramstr, Mimetype, Filesize, Path, Width, Height, Duration)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate cache %s/%s/%s/%s", coll.Name, master.Signature, action, paramstr)
	}
	return cache, nil

}
