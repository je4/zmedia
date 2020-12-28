package media

import (
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/goph/emperror"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type ImageType struct {
	image *vips.ImageRef
}

func NewImageType(reader io.Reader) (*ImageType, error) {
	it := &ImageType{}
	if err := it.LoadImage(reader); err != nil {
		return nil, err
	}
	return it, nil
}

var resizeImageParamRegexp = regexp.MustCompile(`^(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))|(?P<resizeType>(keep|stretch|crop|backgroundblur))|(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))$`)

func (it *ImageType) resize(params []string) (err error) {
	var Width, Height int64
	var Type string = "keep"
	var Format string
	for _, param := range params {
		vals := FindStringSubmatch(resizeImageParamRegexp, strings.ToLower(param))
		for key, val := range vals {
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
		Width = int64(it.image.Width())
		Height = int64(it.image.Height())
	}
	if Width == 0 {
		Width = int64(math.Round(float64(Height) * float64(it.image.Width()) / float64(it.image.Height())))
	}
	if Height == 0 {
		Height = int64(math.Round(float64(Width) * float64(it.image.Height()) / float64(it.image.Width())))
	}

	hScale := float64(Width) / float64(it.image.Width())
	vScale := float64(Height) / float64(it.image.Height())
	var scale float64

	switch Type {
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
		l := (it.image.Width() - int(Width)) / 2
		t := (it.image.Height() - int(Height)) / 2
		if err := it.image.ExtractArea(l, t, int(Width), int(Height)); err != nil {
			return emperror.Wrapf(err, "cannot extract(%v, %v, %v, %v)", l, t, int(Width), int(Height))
		}
	case "backgroundBlur":
		foreground, err := it.image.Copy()
		if err != nil {
			return emperror.Wrap(err, "cannot copy image")
		}
		scale = math.Min(hScale, vScale)
		if err := foreground.Resize(scale, vips.KernelAuto); err != nil {
			return emperror.Wrapf(err, "cannot resize(%v)", scale)
		}
		if err := it.image.GaussianBlur(10); err != nil {
			return emperror.Wrapf(err, "cannot gaussianblur(%v)", 10)
		}
		if err := it.image.BandJoin(foreground); err != nil {
			return emperror.Wrap(err, "cannot bandjoin() images")
		}

	}

	if Type == "keep" {

	}

	it.image.ResizeWithVScale()

	return nil
}

func (it *ImageType) LoadImage(reader io.Reader) error {
	var err error
	it.image, err = vips.NewImageFromReader(reader)
	if err != nil {
		return emperror.Wrapf(err, "cannot read image")
	}
	return nil
}

func (it *ImageType) StoreImage(format string, writer io.Writer) (*CoreMeta, error) {
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
