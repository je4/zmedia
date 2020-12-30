package media

import (
	"fmt"
	"github.com/goph/emperror"
	"gopkg.in/gographics/imagick.v2/imagick"
	"io"
	"strings"
)

type ImageAction struct{}

type ImageType interface {
	LoadImage(reader io.Reader) error
	StoreImage(format string, writer io.Writer) (*CoreMeta, error)
	Resize(params []string) error
	Close()
}

func NewImageAction() (*ImageAction, error) {
	ia := &ImageAction{}
	//	vips.Startup(nil)
	imagick.Initialize()
	return ia, nil
}

func (ia *ImageAction) Close() {
	//	vips.Shutdown()
	imagick.Terminate()
}

func (ia *ImageAction) Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error) {
	var it ImageType
	var err error
	parts := strings.Split(strings.ToLower(meta.Mimetype), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mime type %s", meta.Mimetype)
	}
	if parts[0] != "image" {
		return nil, ErrInvalidType
	}

	it, err = NewImageMagick(reader)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot create image")
	}
	defer it.Close()

	switch action {
	case "resize":
		if err := it.Resize(params); err != nil {
			return nil, emperror.Wrapf(err, "cannot resize image - %v", params)
		}
	default:
		return nil, fmt.Errorf("invalid action %s", action)
	}
	cm, err := it.StoreImage("webp", writer)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot store image")
	}
	return cm, nil
}
