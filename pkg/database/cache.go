package database

type Cache struct {
	db           *MediaDatabase `json:"-"`
	collection   *Collection    `json:"-"`
	master       *Master        `json:"-"`
	Id           int64          `json:"id"`
	CollectionId int64          `json:"collectionid"`
	MasterId     int64          `json:"masterid"`
	Action       string         `json:"action"`
	Params       string         `json:"params"`
	Mimetype     string         `json:"mimetype"`
	Filesize     int64          `json:"filesize"`
	Path         string         `json:"path"`
	Width        int64          `json:"width,omitempty"`
	Height       int64          `json:"height,omitempty"`
	Duration     int64          `json:"duration,omitempty"`
}

func NewCache(db *MediaDatabase, id, collectionid, masterid int64, action string, params string, mimetype string, filesize int64, path string, width, height, duration int64) (*Cache, error) {
	cache := &Cache{
		db:           db,
		Id:           id,
		CollectionId: collectionid,
		MasterId:     masterid,
		Action:       action,
		Params:       params,
		Mimetype:     mimetype,
		Filesize:     filesize,
		Path:         path,
		Width:        width,
		Height:       height,
		Duration:     duration,
	}
	return cache, nil
}

func (c *Cache) GetCollection() (*Collection, error) {

	if c.collection == nil {
		var err error
		c.collection, err = c.db.GetCollectionById(c.CollectionId)
		if err != nil {
			return nil, err
		}
	}
	return c.collection, nil
}

func (c *Cache) GetMaster() (*Master, error) {
	if c.master != nil {
		return c.master, nil
	}
	coll, err := c.GetCollection()
	if err != nil {
		return nil, err
	}
	c.master, err = c.db.GetMasterById(coll, c.MasterId)
	if err != nil {
		return nil, err
	}
	return c.master, nil
}

func (c *Cache) Store() error {
	return c.db.db.StoreCache(c.db, c)
}
