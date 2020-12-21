package database

type Estate struct {
	db          *MediaDatabase `json:"-"`
	Id          int64          `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
}

func NewEstate(mdb *MediaDatabase, id int64, name, description string) (*Estate, error) {
	estate := &Estate{
		db:          mdb,
		Id:          id,
		Name:        name,
		Description: description,
	}
	return estate, nil
}
