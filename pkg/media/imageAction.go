package media

import (
	"fmt"
	"github.com/goph/emperror"
	"github.com/je4/zmedia/v2/pkg/database"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"gopkg.in/gographics/imagick.v3/imagick"
	"io"
	"strconv"
	"strings"
)

type ImageAction struct{}

func (ia *ImageAction) GetType() string {
	return "image"
}

type ImageType interface {
	LoadImage(reader io.Reader) error
	StoreImage(format string) (io.Reader, *CoreMeta, error)
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

/*
var imageParamRegexp = regexp.MustCompile(fmt.Sprintf(`^%s$`, strings.Join([]string{
	`(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))`,
	`(background(?P<background>(none|[0-9a-f]+)))`,
	`(?P<resizeType>(keep|stretch|crop|backgroundblur|extent))`,
	`(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))`,
	`(overlay(?P<overlayCollection>[^-]+)-(?P<overlaySignature>.+))`,
}, "|")))
*/

func buildOptions(params map[string]string) (*ImageOptions, error) {
	var err error
	var io *ImageOptions = &ImageOptions{
		ActionType:   "keep",
		TargetFormat: "png",
	}

	for key, val := range params {
		switch key {
		case "background":
			if val != "none" {
				val = "#" + val
			}
			io.BackgroundColor = val
		case "size":
			sizes := strings.Split(val, "x")
			if sizes[0] != "" {
				if io.Width, err = strconv.ParseInt(sizes[0], 10, 64); err != nil {
					err = emperror.Wrapf(err, "cannot parse width integer %s", val)
					return nil, err
				}
			}
			if sizes[1] != "" {
				if io.Height, err = strconv.ParseInt(sizes[1], 10, 64); err != nil {
					err = emperror.Wrapf(err, "cannot parse height integer %s", val)
					return nil, err
				}
			}
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
	return io, nil
}

func (ia *ImageAction) Do(master *database.Master, action string, params map[string]string, bucket, path string, reader io.Reader) (*CoreMeta, error) {
	var err error
	var it ImageType
	parts := strings.Split(strings.ToLower(master.Mimetype), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mime type %s", master.Mimetype)
	}
	if parts[0] != "image" {
		return nil, ErrInvalidType
	}

	options, err := buildOptions(params)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot build options from param %v", params)
	}

	switch master.Mimetype {
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

	reader, cm, err := it.StoreImage(options.TargetFormat)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot store image %v/%s", master.CollectionId, master.Signature)
	}
	coll, err := master.GetCollection()
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get collection of %v/%s", master.CollectionId, master.Signature)
	}
	stor, err := coll.GetStorage()
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get storage of collection %v", coll.Name)
	}

	if err := stor.Fs.FileWrite(bucket, path, reader, cm.Size, filesystem.FilePutOptions{}); err != nil {
		return nil, emperror.Wrapf(err, "cannot write content to %s/%s/%s", stor.Fs.String(), bucket, path)
	}

	return cm, nil
}
