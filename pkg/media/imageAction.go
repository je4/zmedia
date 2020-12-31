package media

import (
	"fmt"
	"github.com/goph/emperror"
	"gopkg.in/gographics/imagick.v3/imagick"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type ImageAction struct{}

func (ia *ImageAction) GetType() string {
	return "image"
}

type ImageType interface {
	LoadImage(reader io.Reader) error
	StoreImage(format string, writer io.Writer) (*CoreMeta, error)
	Resize(width, height int64, _type, format string) error
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

func (im *ImageMagickV3) GetType() string { return "image" }

var resizeImageParamRegexp = regexp.MustCompile(`^(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))|(?P<resizeType>(keep|stretch|crop|backgroundblur))|(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))$`)

func _getResizeParams(params []string) (Width, Height int64, Type, Format string, err error) {
	Type = "keep"
	for _, param := range params {
		vals := FindStringSubmatch(resizeImageParamRegexp, strings.ToLower(param))
		for key, val := range vals {
			if val == "" {
				continue
			}
			switch key {
			case "sizeWidth":
				if Width, err = strconv.ParseInt(val, 10, 64); err != nil {
					err = emperror.Wrapf(err, "cannot parse integer %s", val)
					return
				}
			case "sizeHeight":
				if Height, err = strconv.ParseInt(val, 10, 64); err != nil {
					err = emperror.Wrapf(err, "cannot parse integer %s", val)
					return
				}
			case "resizeType":
				Type = val
			case "format":
				Format = val
			}
		}
	}
	return
}

func (ia *ImageAction) Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error) {
	var err error
	var it ImageType
	parts := strings.Split(strings.ToLower(meta.Mimetype), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mime type %s", meta.Mimetype)
	}
	if parts[0] != "image" {
		return nil, ErrInvalidType
	}

	switch action {
	case "resize":
		Width, Height, Type, Format, err := _getResizeParams(params)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot evaluate resize parameters")
		}
		switch Type {
		case "keep":
			it, err = NewImageVips(reader)
		case "stretch":
			it, err = NewImageVips(reader)
		case "crop":
			it, err = NewImageVips(reader)
		default:
			it, err = NewImageMagickV3(reader)
		}
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot create image")
		}
		defer it.Close()
		if err := it.Resize(Width, Height, Type, Format); err != nil {
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
