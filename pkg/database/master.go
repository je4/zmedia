package database

type Master struct {
	db           *Database   `json:"-"`
	coll         *Collection `json:"-"`
	Id           string      `json:"id"`
	ParentId     string      `json:"parentid,omitempty"`
	CollectionId string      `json:"collectionid"`
	Urn          string      `json:"urn"`
	Type         string      `json:"type,omitempty"`
	Subtype      string      `json:"subtype,omitempty"`
	Mimetype     string      `json:"mimetype,omitempty"`
	Objecttype   string      `json:"objecttype,omitempty"`
	Sha256       string      `json:"sha256"`
	Metadata     interface{} `json:"metadata,omitempty"`
}

func NewMaster(id, urn, _type, subtype, mimetype, objecttype, sha256 string, metadata interface{}) *Master {
	master := &Master{
		Id:         id,
		Urn:        urn,
		Type:       _type,
		Subtype:    subtype,
		Mimetype:   mimetype,
		Objecttype: objecttype,
		Sha256:     sha256,
		Metadata:   metadata,
	}
	return master
}

func (mas *Master) GetKey() string {
	return MASTER_PREFIX + mas.Id
}
