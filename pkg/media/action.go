package media

import (
	"errors"
	"fmt"
	"github.com/je4/zmedia/v2/pkg/database"
	"io"
)

type CoreMeta struct {
	Width    int64
	Height   int64
	Duration int64
	Mimetype string
	Format   string
	Size     int64
}

var ErrInvalidType = errors.New("mediatype: invalid type")
var ErrInvalidOperation = errors.New("mediatype: invalid operation")

type Action interface {
	Do(master *database.Master, action string, params map[string]string, bucket, path string, reader io.Reader) (*CoreMeta, error)
	Close()
	GetType() string
}

type GenericAction struct{}

func (ga *GenericAction) Do(meta *CoreMeta, action string, params map[string]string, filename string, reader io.Reader) (*CoreMeta, error) {
	if action != "master" {
		return nil, fmt.Errorf("generic action %s not possible", action)
	}
	return nil, fmt.Errorf("generic action not implemented")
}

func (ga *GenericAction) Close() {
}

func (ga *GenericAction) GetType() string {
	return "generic"
}
