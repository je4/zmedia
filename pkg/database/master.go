package database

type Master struct {
	db           *MediaDatabase `json:"-"`
	collection   *Collection    `json:"-"`
	Id           int64          `json:"id"`
	ParentId     int64          `json:"parentid,omitempty"`
	Signature    string         `json:"signature"`
	CollectionId int64          `json:"collectionid"`
	Urn          string         `json:"urn"`
	Type         string         `json:"type,omitempty"`
	Subtype      string         `json:"subtype,omitempty"`
	Mimetype     string         `json:"mimetype,omitempty"`
	Objecttype   string         `json:"objecttype,omitempty"`
	Sha256       string         `json:"sha256"`
	Metadata     interface{}    `json:"metadata,omitempty"`
}

func NewMaster(mdb *MediaDatabase, coll *Collection, id, parentid int64, signature, urn, _type, subtype, mimetype, objecttype, sha256 string, metadata interface{}) (*Master, error) {
	master := &Master{
		db:           mdb,
		collection:   coll,
		Id:           id,
		ParentId:     parentid,
		CollectionId: coll.Id,
		Urn:          urn,
		Type:         _type,
		Subtype:      subtype,
		Mimetype:     mimetype,
		Objecttype:   objecttype,
		Sha256:       sha256,
		Metadata:     metadata,
	}
	return master, nil
}

func (m *Master) GetCollection() (*Collection, error) {
	if m.collection == nil {
		var err error
		m.collection, err = m.db.GetCollectionById(m.Id)
		if err != nil {
			return nil, err
		}
	}
	return m.collection, nil
}
