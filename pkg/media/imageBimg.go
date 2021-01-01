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

type ImageBimg struct {
	image *bimg.Image
	buf   []byte
	meta  bimg.ImageMetadata
}

func NewImageBimg(reader io.Reader) (*ImageBimg, error) {
	it := &ImageBimg{}
	if err := it.LoadImage(reader); err != nil {
		return nil, err
	}
	return it, nil
}

var resizeImageBimgParamRegexp = regexp.MustCompile(`^(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))|(?P<resizeType>(keep|stretch|crop|backgroundblur))|(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))$`)

func (ib *ImageBimg) Resize(params []string) (err error) {
	var Width, Height int64
	var Type string = "keep"
	var Format string
	for _, param := range params {
		vals := FindStringSubmatch(resizeImageBimgParamRegexp, strings.ToLower(param))
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
		Width = int64(ib.meta.Size.Width)
		Height = int64(ib.meta.Size.Height)
	}
	if Width == 0 {
		Width = int64(math.Round(float64(Height) * float64(ib.meta.Size.Height) / float64(ib.meta.Size.Height)))
	}
	if Height == 0 {
		Height = int64(math.Round(float64(Width) * float64(ib.meta.Size.Width) / float64(ib.meta.Size.Width)))
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
		w, h := CalcSizeMin(int64(ib.meta.Size.Width), int64(ib.meta.Size.Height), Width, Height)
		options.Width = int(w)
		options.Height = int(h)
		options.Embed = true
	case "stretch":
		options.Width = int(Width)
		options.Height = int(Height)
		options.Force = true
	case "crop":
		w, h := CalcSizeMin(int64(ib.meta.Size.Width), int64(ib.meta.Size.Height), (Width), (Height))
		options.Width = int(w)
		options.Height = int(h)
		options.Embed = true
		options.Crop = true
	case "backgroundblur":
		options.Width = int(Width)
		options.Height = int(Height)
		options.Force = true
		options.GaussianBlur = bimg.GaussianBlur{Sigma: 10}

		foreground := bimg.NewImage(ib.buf)
		w, h := CalcSizeMin(int64(ib.meta.Size.Width), int64(ib.meta.Size.Height), (Width), (Height))
		fgOptions := bimg.Options{
			Height: int(h),
			Width:  int(w),
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

	if ib.buf, err = ib.image.Process(options); err != nil {
		return emperror.Wrapf(err, "cannot process image - %v", options)
	}
	ib.image = bimg.NewImage(ib.buf)
	if ib.meta, err = ib.image.Metadata(); err != nil {
		return emperror.Wrapf(err, "cannot get final metadata")
	}
	return nil
}

func (ib *ImageBimg) LoadImage(reader io.Reader) error {
	var err error
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return emperror.Wrapf(err, "cannot read image content")
	}
	ib.buf = buf.Bytes()
	ib.image = bimg.NewImage(ib.buf)
	if ib.meta, err = ib.image.Metadata(); err != nil {
		return emperror.Wrap(err, "cannot get metadata from image")
	}

	return nil
}

func (ib *ImageBimg) StoreImage(writer io.Writer) (*CoreMeta, error) {
	num, err := writer.Write(ib.buf)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot write data")
	}
	if num == 0 {
		return nil, fmt.Errorf("zero bytes written")
	}

	cm := &CoreMeta{
		Width:    int64(ib.meta.Size.Width),
		Height:   int64(ib.meta.Size.Height),
		Duration: 0,
		Format:   ib.meta.Type,
	}

	switch ib.meta.Type {
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
		return nil, fmt.Errorf("invalid image type %s", ib.meta.Type)
	}

	return cm, nil
}
