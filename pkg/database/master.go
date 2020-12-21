package database

type Master struct {
	db           *MediaDatabase `json:"-"`
	coll         *Collection    `json:"-"`
	Id           int64          `json:"id"`
	ParentId     int64          `json:"parentid,omitempty"`
	CollectionId int64          `json:"collectionid"`
	Urn          string         `json:"urn"`
	Type         string         `json:"type,omitempty"`
	Subtype      string         `json:"subtype,omitempty"`
	Mimetype     string         `json:"mimetype,omitempty"`
	Objecttype   string         `json:"objecttype,omitempty"`
	Sha256       string         `json:"sha256"`
	Metadata     interface{}    `json:"metadata,omitempty"`
}

func NewMaster(mdb *MediaDatabase, id int64, urn, _type, subtype, mimetype, objecttype, sha256 string, metadata interface{}) (*Master, error) {
	master := &Master{
		db:         mdb,
		Id:         id,
		Urn:        urn,
		Type:       _type,
		Subtype:    subtype,
		Mimetype:   mimetype,
		Objecttype: objecttype,
		Sha256:     sha256,
		Metadata:   metadata,
	}
	return master, nil
}
