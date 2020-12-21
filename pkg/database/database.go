package database

type Database interface {
	LoadEstates(mdb *MediaDatabase, callback func(estate *Estate) error) error
	LoadStorages(callback func(storage *Storage) error) error
	LoadCollections(mdb *MediaDatabase, callback func(storage *Collection) error) error
	GetMaster(collection *Collection, signature string) (*Master, error)
}
