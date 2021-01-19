package indexer

import ffmpeg_models "github.com/je4/goffmpeg/models"

type FFProbe struct {
	command string
}

var metadata ffmpeg_models.Metadata

func NewFFProbe(command string) (*FFProbe, error) {
	ffp := &FFProbe{
		command: command,
	}
	return ffp, nil
}

func (fp *FFProbe) Identify(urlstring string) (map[string]interface{}, error) {
	return nil, nil
}
