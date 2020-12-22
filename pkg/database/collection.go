package database

type Collection struct {
	db          *MediaDatabase `json:"-"`
	estate      *Estate        `json:"-"`
	storage     *Storage       `json:"-"`
	Id          int64          `json:"id"`
	EstateId    int64          `json:"estateid"`
	StorageId   int64          `json:"storageid"`
	Name        string         `json:"string"`
	Description string         `json:"description,omitempty"`
	ZoteroGroup int64          `json:"zoterogroup,omitempty"`
}

func NewCollection(mdb *MediaDatabase, id int64, storage *Storage, estate *Estate, name, description string, zoteroGroup int64) (*Collection, error) {
	coll := &Collection{
		db:          mdb,
		Id:          id,
		storage:     storage,
		estate:      estate,
		StorageId:   storage.Id,
		EstateId:    estate.Id,
		Name:        name,
		Description: description,
		ZoteroGroup: zoteroGroup,
	}
	return coll, nil
}

func (coll *Collection) GetStorage() (*Storage, error) {
	if coll.storage == nil {
		var err error
		coll.storage, err = coll.db.GetStorageById(coll.StorageId)
		if err != nil {
			return nil, err
		}
	}
	return coll.storage, nil
}

func (coll *Collection) GetEstate() (*Estate, error) {
	if coll.estate == nil {
		var err error
		coll.estate, err = coll.db.GetEstateById(coll.EstateId)
		if err != nil {
			return nil, err
		}
	}
	return coll.estate, nil
}

func (coll *Collection) GetMaster(signature string) (*Master, error) {
	return coll.db.GetMaster(coll, signature)
}

func (coll *Collection) GetMasterById(masterid int64) (*Master, error) {
	return coll.db.GetMasterById(coll, masterid)
}
