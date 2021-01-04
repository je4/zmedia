package media

import (
	"errors"
	"io"
)

type MediaAction interface {
	Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error)
}

type CoreMeta struct {
	Width    int64
	Height   int64
	Duration int64
	Mimetype string
	Format   string
}

var ErrInvalidType = errors.New("mediatype: invalid type")
var ErrInvalidOperation = errors.New("mediatype: invalid operation")
