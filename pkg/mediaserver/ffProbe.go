package mediaserver

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/goph/emperror"
	ffmpeg_models "github.com/je4/goffmpeg/models"
	"os/exec"
	"strconv"
	"time"
)

type FFProbe struct {
	ffprobe string
	mh      *MediaHandler
}

var metadata ffmpeg_models.Metadata

func NewFFProbe(mh *MediaHandler, command string) (*FFProbe, error) {
	ffp := &FFProbe{
		ffprobe: command, mh: mh,
	}
	return ffp, nil
}

func (fp *FFProbe) SetMediaHandler(mh *MediaHandler) {
	fp.mh = mh
}

func (fp *FFProbe) GetMetadata(filename string, timeout time.Duration) (width, height, duration int64, mimetype, sub string, metadata interface{}, err error) {
	var ffmeta ffmpeg_models.Metadata

	fs, bucket, path, err := fp.mh.GetFS(filename)
	if err != nil {
		err = emperror.Wrapf(err, "cannot get filesystem for %s", filename)
		return
	}

	url, err := fs.GETUrl(bucket, path, timeout)
	if err != nil {
		err = emperror.Wrapf(err, "cannot get url for %s", filename)
		return
	}

	var fname string
	if fs.IsLocal() {
		fname = url.Path
	} else {
		fname = url.String()
	}

	cmdparam := []string{
		"-i", fname,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_error",
	}
	cmdfile := fp.ffprobe

	var out, errb bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdfile, cmdparam...)
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err = cmd.Run()
	if err != nil {
		err = emperror.Wrapf(err, "error executing (%s %s): %v %v", cmdfile, cmdparam, out.String(), errb.String())
		return
	}

	if err = json.Unmarshal([]byte(out.String()), &ffmeta); err != nil {
		err = emperror.Wrapf(err, "cannot unmarshall metadata: %s", out.String())
		return
	}

	// calculate duration and dimension
	d, _ := strconv.ParseFloat(ffmeta.Format.Duration, 64)
	duration = int64(time.Second * time.Duration(d))
	for _, stream := range ffmeta.Streams {
		if stream.Width > 0 || stream.Height > 0 {
			width = int64(stream.Width)
			height = int64(stream.Height)
		}
	}
	metadata = ffmeta
	return
}
