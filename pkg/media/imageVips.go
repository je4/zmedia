package media

import (
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/goph/emperror"
	"io"
	"math"
	"regexp"
)

type ImageVips struct {
	image *vips.ImageRef
}

func NewImageVips(reader io.Reader) (*ImageVips, error) {
	it := &ImageVips{}
	if err := it.LoadImage(reader); err != nil {
		return nil, err
	}
	return it, nil
}

var resizeImageVipsParamRegexp = regexp.MustCompile(`^(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))|(?P<resizeType>(keep|stretch|crop|backgroundblur))|(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))$`)

func (it *ImageVips) Close() {}

func (im *ImageVips) GetType() string { return "image" }

func (it *ImageVips) Resize(width, height int64, _type, format string) (err error) {

	//
	// calculate missing size parameter
	//
	if width == 0 && height == 0 {
		width = int64(it.image.Width())
		height = int64(it.image.Height())
	}
	if width == 0 {
		width = int64(math.Round(float64(height) * float64(it.image.Width()) / float64(it.image.Height())))
	}
	if height == 0 {
		height = int64(math.Round(float64(width) * float64(it.image.Height()) / float64(it.image.Width())))
	}

	if err := it.image.AutoRotate(); err != nil {
		return emperror.Wrapf(err, "cannot autorotate image")
	}

	hScale := float64(width) / float64(it.image.Width())
	vScale := float64(height) / float64(it.image.Height())
	var scale float64

	switch _type {
	case "keep":
		scale = math.Min(hScale, vScale)
		if err := it.image.Resize(scale, vips.KernelAuto); err != nil {
			return emperror.Wrapf(err, "cannot resize(%v)", scale)
		}
	case "stretch":
		if it.image.ResizeWithVScale(hScale, vScale, vips.KernelAuto); err != nil {
			return emperror.Wrapf(err, "cannot resize(%v, %v)", hScale, vScale)
		}
	case "crop":
		scale = math.Max(hScale, vScale)
		if err := it.image.Resize(scale, vips.KernelAuto); err != nil {
			return emperror.Wrapf(err, "cannot resize(%v)", scale)
		}
		l := (it.image.Width() - int(width)) / 2
		t := (it.image.Height() - int(height)) / 2
		if err := it.image.ExtractArea(l, t, int(width), int(height)); err != nil {
			return emperror.Wrapf(err, "cannot extract(%v, %v, %v, %v)", l, t, int(width), int(height))
		}
	}

	return nil
}

func (it *ImageVips) LoadImage(reader io.Reader) error {
	var err error
	it.image, err = vips.NewImageFromReader(reader)
	if err != nil {
		return emperror.Wrapf(err, "cannot read image")
	}
	return nil
}

func (it *ImageVips) StoreImage(format string, writer io.Writer) (*CoreMeta, error) {
	var ep *vips.ExportParams
	var mimetype string
	switch format {
	case "jpeg":
		ep = vips.NewDefaultJPEGExportParams()
		mimetype = "image/jpeg"
	case "png":
		ep = vips.NewDefaultPNGExportParams()
		mimetype = "image/png"
	case "webp":
		ep = vips.NewDefaultWEBPExportParams()
		mimetype = "image/webp"
	default:
		return nil, fmt.Errorf("invalid format %s", format)
	}
	bytes, meta, err := it.image.Export(ep)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot export to %s", format)
	}
	num, err := writer.Write(bytes)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot write data")
	}
	if num == 0 {
		return nil, fmt.Errorf("zero bytes written")
	}
	cm := &CoreMeta{
		Width:    int64(meta.Width),
		Height:   int64(meta.Height),
		Duration: 0,
		Format:   format,
		Mimetype: mimetype,
	}
	return cm, nil
}
