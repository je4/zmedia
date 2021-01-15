package indexer

import (
	ffmpeg_models "github.com/je4/goffmpeg/models"
	"io"
)

type FFProbe struct {
	command string
}

var metadata ffmpeg_models.Metadata

func (fp *FFProbe) Index(reader io.Reader) (map[string]interface{}, error) {

}
