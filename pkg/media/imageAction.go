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
	Resize(options *ImageOptions) error
	Close()
}

type ImageOptions struct {
	Width, Height                       int64
	ActionType                          string
	TargetFormat                        string
	OverlayCollection, OverlaySignature string
	BackgroundColor                     string
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

var imageParamRegexp = regexp.MustCompile(fmt.Sprintf(`^%s$`, strings.Join([]string{
	`(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))`,
	`(background(?P<background>(none|[0-9a-f]+)))`,
	`(?P<resizeType>(keep|stretch|crop|backgroundblur|extent))`,
	`(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))`,
	`(overlay(?P<overlayCollection>[^-]+)-(?P<overlaySignature>.+))`,
}, "|")))

func buildOptions(params []string) (*ImageOptions, error) {
	var err error
	var io *ImageOptions = &ImageOptions{
		ActionType:   "keep",
		TargetFormat: "keep",
	}

	for _, param := range params {
		vals := FindStringSubmatch(imageParamRegexp, strings.ToLower(param))
		for key, val := range vals {
			if val == "" {
				continue
			}
			switch key {
			case "background":
				if val != "none" {
					val = "#" + val
				}
				io.BackgroundColor = val
			case "sizeWidth":
				if io.Width, err = strconv.ParseInt(val, 10, 64); err != nil {
					err = emperror.Wrapf(err, "cannot parse integer %s", val)
					return nil, err
				}
			case "sizeHeight":
				if io.Height, err = strconv.ParseInt(val, 10, 64); err != nil {
					err = emperror.Wrapf(err, "cannot parse integer %s", val)
					return nil, err
				}
			case "resizeType":
				io.ActionType = val
			case "format":
				io.TargetFormat = val
			case "overlayCollection":
				io.OverlayCollection = val
			case "overlaySignature":
				io.OverlaySignature = val
			}
		}
	}

	return io, nil
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

	options, err := buildOptions(params)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot build options from param %v", params)
	}

	switch meta.Mimetype {
	case "image/gif":
		it, err = NewImageMagickV3(reader)
	default:
		switch action {
		case "resize":
			switch options.ActionType {
			case "keep":
				it, err = NewImageVips(reader)
			case "stretch":
				it, err = NewImageVips(reader)
			case "crop":
				it, err = NewImageVips(reader)
			default:
				it, err = NewImageMagickV3(reader)
			}
		default:
			return nil, fmt.Errorf("invalid action %s", action)
		}
	}
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot create image")
	}
	defer it.Close()
	switch action {
	case "resize":
		if err := it.Resize(options); err != nil {
			return nil, emperror.Wrapf(err, "cannot resize image - %v", params)
		}
	default:
		return nil, fmt.Errorf("invalid action - %s", action)
	}

	cm, err := it.StoreImage(options.TargetFormat, writer)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot store image")
	}
	return cm, nil
}
