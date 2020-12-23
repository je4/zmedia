package media

import (
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/goph/emperror"
	"io"
	"strings"
)

type ImageType struct {
	image *vips.ImageRef
}

func NewImageType() (*ImageType, error) {
	it := &ImageType{}
	return it, nil
}

func (it *ImageType) MimeOK(mimetype string) bool {
	parts := strings.Split(strings.ToLower(mimetype), "/")
	if len(parts) != 2 {
		return false
	}
	ok := parts[0] == "image"

	return ok
}

func (it *ImageType) Do(action string, params []string, reader io.Reader, writer io.Writer) error {
	if err := it.LoadImage(reader); err != nil {
		return emperror.Wrapf(err, "cannot read image")
	}

}

func (it *ImageType) resize(params []string) error {

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
