package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	"os"
	"os/exec"
	"time"
)

type ImagickIdentify struct {
	identify string
}

func NewImagickIdentify(identify string) (*ImagickIdentify, error) {
	ii := &ImagickIdentify{identify: identify}
	return ii, nil
}

func (ii *ImagickIdentify) GetMetadata(urlstring string, timeout time.Duration) (width, height, duration int64, mimetype, sub string, metadata interface{}, err error) {
	var md = make(map[string]interface{})
	var metadataInt interface{}

	// {"width":%w,"height":%h,"images":%n,"magick":"%m","orientation":"%[orientation]"}
	cmdparam := []string{
		"-format", `{"width":%w,"height":%h,"images":%n,"magick":"%m","orientation":"%[orientation]"}`,
		"-define", fmt.Sprintf("registry:temporary-path='%s'", os.TempDir()),
		urlstring,
	}

	var out, errb bytes.Buffer
	out.Grow(1024 * 1024) // 1MB size
	errb.Grow(1024 * 1024)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ii.identify, cmdparam...)
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err = cmd.Run()
	if err != nil {
		err = emperror.Wrapf(err, "error executing (%s %s): %v - %v", ii.identify, cmdparam, out.String(), errb.String())
		return
	}

	if err = json.Unmarshal([]byte(out.String()), &metadataInt); err != nil {
		err = emperror.Wrapf(err, "cannot unmarshall metadata: %s", out.String())
		return
	}

	switch val := metadataInt.(type) {
	case []interface{}:
		// todo: check for content and type
		if len(val) != 1 {
			err = fmt.Errorf("wrong number of objects in image magick result list - %v", len(val))
			return
		}
		var ok bool
		md, ok = val[0].(map[string]interface{})
		if !ok {
			err = fmt.Errorf("wrong object type in image magick result - %T", val[0])
			return
		}
	case map[string]interface{}:
		md = val
	default:
		err = fmt.Errorf("invalid return type from image magick - %T", val)
		return
	}

	_image, ok := md["image"]
	if !ok {
		err = emperror.Wrapf(err, "no image field in %s", out.String())
		return
	}
	// calculate mimetype and dimensions
	image, ok := _image.(map[string]interface{})
	mimetype, ok = image["mimeType"].(string)
	_geometry, ok := image["geometry"].(map[string]interface{})
	if ok {
		w, ok := _geometry["width"].(float64)
		if ok {
			width = int64(w)
		}
		h, ok := _geometry["height"].(float64)
		if ok {
			height = int64(h)
		}
	}
	metadata = md
	return

}
