package media

import (
	"bytes"
	"fmt"
	"github.com/goph/emperror"
	"github.com/h2non/bimg"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type ImageType struct {
	image *bimg.Image
	buf   []byte
	meta  bimg.ImageMetadata
}

func NewImageType(reader io.Reader) (*ImageType, error) {
	it := &ImageType{}
	if err := it.LoadImage(reader); err != nil {
		return nil, err
	}
	return it, nil
}

var resizeImageParamRegexp = regexp.MustCompile(`^(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))|(?P<resizeType>(keep|stretch|crop|backgroundblur))|(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))$`)

func (it *ImageType) Resize(params []string) (err error) {
	var Width, Height int64
	var Type string = "keep"
	var Format string
	for _, param := range params {
		vals := FindStringSubmatch(resizeImageParamRegexp, strings.ToLower(param))
		for key, val := range vals {
			if val == "" {
				continue
			}
			switch key {
			case "sizeWidth":
				if Width, err = strconv.ParseInt(val, 10, 64); err != nil {
					return emperror.Wrapf(err, "cannot parse integer %s", val)
				}
			case "sizeHeight":
				if Height, err = strconv.ParseInt(val, 10, 64); err != nil {
					return emperror.Wrapf(err, "cannot parse integer %s", val)
				}
			case "resizeType":
				Type = val
			case "format":
				Format = val
			}
		}
	}

	//
	// calculate missing size parameter
	//
	if Width == 0 && Height == 0 {
		Width = int64(it.meta.Size.Width)
		Height = int64(it.meta.Size.Height)
	}
	if Width == 0 {
		Width = int64(math.Round(float64(Height) * float64(it.meta.Size.Height) / float64(it.meta.Size.Height)))
	}
	if Height == 0 {
		Height = int64(math.Round(float64(Width) * float64(it.meta.Size.Width) / float64(it.meta.Size.Width)))
	}

	var options bimg.Options

	switch Format {
	case "jpeg":
		options.Type = bimg.JPEG
	case "png":
		options.Type = bimg.PNG
	case "webp":
		options.Type = bimg.WEBP
	case "ptiff":
		options.Type = bimg.TIFF
	default:
		return fmt.Errorf("invalid format %s", Format)
	}

	switch Type {
	case "keep":
		w, h := CalcSize(it.meta.Size.Width, it.meta.Size.Height, int(Width), int(Height))
		options.Width = w
		options.Height = h
		options.Embed = true
	case "stretch":
		options.Width = int(Width)
		options.Height = int(Height)
		options.Force = true
	case "crop":
		w, h := CalcSize(it.meta.Size.Width, it.meta.Size.Height, int(Width), int(Height))
		options.Width = w
		options.Height = h
		options.Embed = true
		options.Crop = true
	case "backgroundblur":
		options.Width = int(Width)
		options.Height = int(Height)
		options.Force = true
		options.GaussianBlur = bimg.GaussianBlur{Sigma: 10}

		foreground := bimg.NewImage(it.buf)
		w, h := CalcSize(it.meta.Size.Width, it.meta.Size.Height, int(Width), int(Height))
		fgOptions := bimg.Options{
			Height: h,
			Width:  w,
			Embed:  true,
		}
		foregroundBytes, err := foreground.Process(fgOptions)
		if err != nil {
			return emperror.Wrapf(err, "cannot resize(%v, %v)", int(Width), int(Height))
		}
		foregroundMeta, err := foreground.Metadata()
		if err != nil {
			return emperror.Wrap(err, "cannot get metadata from foreground image")
		}
		options.WatermarkImage = bimg.WatermarkImage{
			Left:    (int(Width) - foregroundMeta.Size.Width) / 2,
			Top:     (int(Height) - foregroundMeta.Size.Height) / 2,
			Buf:     foregroundBytes,
			Opacity: 0,
		}

	}

	if it.buf, err = it.image.Process(options); err != nil {
		return emperror.Wrapf(err, "cannot process image - %v", options)
	}
	it.image = bimg.NewImage(it.buf)
	if it.meta, err = it.image.Metadata(); err != nil {
		return emperror.Wrapf(err, "cannot get final metadata")
	}
	return nil
}

func (it *ImageType) LoadImage(reader io.Reader) error {
	var err error
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return emperror.Wrapf(err, "cannot read image content")
	}
	it.buf = buf.Bytes()
	it.image = bimg.NewImage(it.buf)
	if it.meta, err = it.image.Metadata(); err != nil {
		return emperror.Wrap(err, "cannot get metadata from image")
	}

	return nil
}

func (it *ImageType) StoreImage(writer io.Writer) (*CoreMeta, error) {
	num, err := writer.Write(it.buf)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot write data")
	}
	if num == 0 {
		return nil, fmt.Errorf("zero bytes written")
	}

	cm := &CoreMeta{
		Width:    int64(it.meta.Size.Width),
		Height:   int64(it.meta.Size.Height),
		Duration: 0,
		Format:   it.meta.Type,
	}

	switch it.meta.Type {
	case "jpeg":
		cm.Mimetype = "image/jpeg"
	case "png":
		cm.Mimetype = "image/png"
	case "webp":
		cm.Mimetype = "image/webp"
	case "tiff":
		cm.Mimetype = "image/tigg"
	case "gif":
		cm.Mimetype = "image/gif"
	case "pdf":
		cm.Mimetype = "application/pdf"
	case "svg":
		cm.Mimetype = "image/svg"
		//	case "magick":
	case "heif":
		cm.Mimetype = "image/heif"
	case "avif":
		cm.Mimetype = "image/avif"
	default:
		return nil, fmt.Errorf("invalid image type %s", it.meta.Type)
	}

	return cm, nil
}
