package media

import (
	"fmt"
	"github.com/goph/emperror"
	"io"
	"strings"
)

type ImageAction struct{}

func (ia *ImageAction) Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error) {
	parts := strings.Split(strings.ToLower(meta.Mimetype), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mime type %s", meta.Mimetype)
	}
	if parts[0] != "image" {
		return nil, ErrInvalidType
	}

	it, err := NewImageType(reader)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot create image")
	}

	switch action {
	case "resize":
		if err := it.Resize(params); err != nil {
			return nil, emperror.Wrapf(err, "cannot resize image - %v", params)
		}
	default:
		return nil, fmt.Errorf("invalid action %s", action)
	}
	cm, err := it.StoreImage(writer)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot store image")
	}
	return cm, nil
}
