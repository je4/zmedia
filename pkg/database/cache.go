package database

type Cache struct {
	db       *MediaDatabase `json:"-"`
	master   *Master        `json:"-"`
	Id       int64          `json:"id"`
	MasterId int64          `json:"masterid"`
	Action   string         `json:"action"`
	Params   string         `json:"params"`
	Mimetype string         `json:"mimetype"`
	Filesize int64          `json:"filesize"`
	Path     string         `json:"path"`
	Width    int64          `json:"width,omitempty"`
	Height   int64          `json:"height,omitempty"`
	Duration int64          `json:"duration,omitempty"`
}

func NewCache(db *MediaDatabase, id, masterid int64, action string, params string, mimetype string, filesize int64, path string, width, height, duration int64) (*Cache, error) {
	cache := &Cache{
		db:       db,
		Id:       id,
		MasterId: masterid,
		Action:   action,
		Params:   params,
		Mimetype: mimetype,
		Filesize: filesize,
		Path:     path,
		Width:    width,
		Height:   height,
		Duration: duration,
	}
	return cache, nil
}

func (c *Cache) GetMaster() (*Master, error) {
	if c.master == nil {
		var err error
		c.master, err = c.db.GetMasterById()
	}
}
