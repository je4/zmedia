package media

import (
	"bytes"
	"github.com/goph/emperror"
	"gopkg.in/gographics/imagick.v2/imagick"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type ImageMagick struct {
	mw *imagick.MagickWand
}

func NewImageMagick(reader io.Reader) (*ImageMagick, error) {
	im := &ImageMagick{mw: imagick.NewMagickWand()}
	if err := im.LoadImage(reader); err != nil {
		return nil, err
	}
	return im, nil
}

func (im *ImageMagick) Close() {
	im.mw.Destroy()
}

func (im *ImageMagick) LoadImage(reader io.Reader) error {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return emperror.Wrapf(err, "cannot read raw image blob")
	}
	if err := im.mw.ReadImageBlob(buf.Bytes()); err != nil {
		return emperror.Wrapf(err, "cannot read image from blob")
	}
	return nil
}

func (im *ImageMagick) StoreImage(format string, writer io.Writer) (*CoreMeta, error) {
	if err := im.mw.SetFilename(format); err != nil {
		return nil, emperror.Wrapf(err, "cannot set format %s", format)
	}
	buf := im.mw.GetImagesBlob()
	if _, err := writer.Write(buf); err != nil {
		return nil, emperror.Wrapf(err, "cannot write raw image data")
	}

	cm := &CoreMeta{
		Width:    int64(im.mw.GetImageWidth()),
		Height:   int64(im.mw.GetImageHeight()),
		Duration: 0,
		Format:   im.mw.GetFormat(),
		Mimetype: "application/octet-stream",
	}
	return cm, nil
}

var resizeImageMagickParamRegexp = regexp.MustCompile(`^(size(?P<sizeWidth>[0-9]*)x(?P<sizeHeight>[0-9]*))|(?P<resizeType>(keep|stretch|crop|backgroundblur))|(format(?P<format>jpeg|webp|png|gif|ptiff|jpeg2000))$`)

func (im *ImageMagick) Resize(params []string) error {
	var err error
	var Width, Height int64
	var Type string = "keep"
	//	var Format string
	for _, param := range params {
		vals := FindStringSubmatch(resizeImageMagickParamRegexp, strings.ToLower(param))
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
				//				Format = val
			}
		}
	}

	//
	// calculate missing size parameter
	//
	if Width == 0 && Height == 0 {
		Width = int64(im.mw.GetImageWidth())
		Height = int64(im.mw.GetImageHeight())
	}
	if Width == 0 {
		Width = int64(math.Round(float64(Height) * float64(im.mw.GetImageWidth()) / float64(im.mw.GetImageHeight())))
	}
	if Height == 0 {
		Height = int64(math.Round(float64(Width) * float64(im.mw.GetImageHeight()) / float64(im.mw.GetImageWidth())))
	}

	if err := im.mw.AutoOrientImage(); err != nil {
		return emperror.Wrapf(err, "cannot auto orient image")
	}

	switch Type {
	case "keep":
	case "stretch":
	case "crop":
	case "backgroundblur":

		foreground := im.mw.Clone()
		nw, nh := CalcSize(int64(im.mw.GetImageWidth()), int64(im.mw.GetImageHeight()), int64(Width), int64(Height))
		if err := foreground.ResizeImage(uint(nw), uint(nh), imagick.FILTER_LANCZOS, 1); err != nil {
			return emperror.Wrapf(err, "cannot resizeimage(%v, %v) - foreground", uint(nw), uint(nh))
		}

		if err := im.mw.ResizeImage(uint(Width), uint(Height), imagick.FILTER_LANCZOS, 1); err != nil {
			return emperror.Wrapf(err, "cannot resizeimage(%v, %v)", uint(Width), uint(Height))
		}

		if err := im.mw.BlurImage(4, 10.0); err != nil {
			return emperror.Wrapf(err, "cannot blurimage(%v, %v)", 4.0, 10.0)
		}

		if err := im.mw.CompositeImageGravity(foreground, imagick.COMPOSITE_OP_COPY, imagick.GRAVITY_CENTER); err != nil {
			return emperror.Wrapf(err, "cannot composite images")
		}
	}
	return nil
}
