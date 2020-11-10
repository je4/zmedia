package database

type Estate struct {
	db          *Database `json:"-"`
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
}

func NewEstate(id, name, description string) *Estate {
	estate := &Estate{
		Id:          id,
		Name:        name,
		Description: description,
	}
	return estate
}

func (es *Estate) GetKey() string {
	return ESTATE_PREFIX + es.Id
}
