package mediaserver

import ffmpeg_models "github.com/je4/goffmpeg/models"

type FFProbe struct {
	command string
	mh      *MediaHandler
}

var metadata ffmpeg_models.Metadata

func NewFFProbe(mh *MediaHandler, command string) (*FFProbe, error) {
	ffp := &FFProbe{
		command: command, mh: mh,
	}
	return ffp, nil
}

func (fp *FFProbe) SetMediaHandler(mh *MediaHandler) {
	fp.mh = mh
}

func (fp *FFProbe) Identify(urlstring string) (map[string]interface{}, error) {
	return nil, nil
}
