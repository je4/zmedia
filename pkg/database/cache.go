package database

type Cache struct {
	db       *Database         `json:"-"`
	Id       string            `json:"id"`
	MasterId string            `json:"masterid"`
	Action   string            `json:"action"`
	Params   map[string]string `json:"params"`
	Mimetype string            `json:"mimetype"`
	Filesize int64             `json:"filesize"`
	Path     string            `json:"path"`
	Width    int64             `json:"width,omitempty"`
	Height   int64             `json:"height,omitempty"`
	Duration int64             `json:"duration,omitempty"`
}

func NewCache(id, action string, params map[string]string, mimetype string, filesize int64, path string, width, height, duration int64) *Cache {
	cache := &Cache{
		Id:       id,
		Action:   action,
		Params:   params,
		Mimetype: mimetype,
		Filesize: filesize,
		Path:     path,
		Width:    width,
		Height:   height,
		Duration: duration,
	}
	return cache
}

func (cache *Cache) GetKey() string {
	return CACHE_PREFIX + cache.Id
}
