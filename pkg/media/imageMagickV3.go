package media

import (
	"bytes"
	"github.com/goph/emperror"
	"gopkg.in/gographics/imagick.v3/imagick"
	"io"
	"math"
)

type ImageMagickV3 struct {
	mw *imagick.MagickWand
}

func NewImageMagickV3(reader io.Reader) (*ImageMagickV3, error) {
	im := &ImageMagickV3{mw: imagick.NewMagickWand()}
	if err := im.LoadImage(reader); err != nil {
		return nil, err
	}
	return im, nil
}

func (im *ImageMagickV3) Close() {
	im.mw.Destroy()
}

func (im *ImageMagickV3) LoadImage(reader io.Reader) error {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return emperror.Wrapf(err, "cannot read raw image blob")
	}
	if err := im.mw.ReadImageBlob(buf.Bytes()); err != nil {
		return emperror.Wrapf(err, "cannot read image from blob")
	}
	return nil
}

func (im *ImageMagickV3) StoreImage(format string, writer io.Writer) (*CoreMeta, error) {
	if err := im.mw.SetFilename(format); err != nil {
		return nil, emperror.Wrapf(err, "cannot set format %s", format)
	}
	buf := im.mw.GetImagesBlob()
	if _, err := io.Copy(writer, bytes.NewBuffer(buf)); err != nil {
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

func (im *ImageMagickV3) Resize(width, height int64, _type, format string) error {
	//
	// calculate missing size parameter
	//
	if width == 0 && height == 0 {
		width = int64(im.mw.GetImageWidth())
		height = int64(im.mw.GetImageHeight())
	}
	if width == 0 {
		width = int64(math.Round(float64(height) * float64(im.mw.GetImageWidth()) / float64(im.mw.GetImageHeight())))
	}
	if height == 0 {
		height = int64(math.Round(float64(width) * float64(im.mw.GetImageHeight()) / float64(im.mw.GetImageWidth())))
	}

	if err := im.mw.AutoOrientImage(); err != nil {
		return emperror.Wrapf(err, "cannot auto orient image")
	}

	switch _type {
	case "keep":
	case "stretch":
	case "crop":
	case "backgroundblur":
		foreground := im.mw.Clone()
		nw, nh := CalcSize(int64(im.mw.GetImageWidth()), int64(im.mw.GetImageHeight()), int64(width), int64(height))
		if err := foreground.ResizeImage(uint(nw), uint(nh), imagick.FILTER_LANCZOS); err != nil {
			return emperror.Wrapf(err, "cannot resizeimage(%v, %v) - foreground", uint(nw), uint(nh))
		}

		if err := im.mw.ResizeImage(uint(width), uint(height), imagick.FILTER_LANCZOS); err != nil {
			return emperror.Wrapf(err, "cannot resizeimage(%v, %v)", uint(width), uint(height))
		}

		if err := im.mw.BlurImage(20, 30.0); err != nil {
			return emperror.Wrapf(err, "cannot blurimage(%v, %v)", 20.0, 30.0)
		}

		if err := im.mw.CompositeImageGravity(foreground, imagick.COMPOSITE_OP_COPY, imagick.GRAVITY_CENTER); err != nil {
			return emperror.Wrapf(err, "cannot composite images")
		}
	}
	return nil
}
