package indexer

import (
	"github.com/goph/emperror"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"regexp"
)

type Indexer struct {
	ffProbe    *FFProbe
	Siegfried  *Siegfried
	tempfolder string
}

func NewIndexer(siegfriedurl, ffprobe, tempfolder string) (*Indexer, error) {
	ffp, err := NewFFProbe(ffprobe)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate ffprobe %s", ffprobe)
	}
	sf, err := NewSiegfried(siegfriedurl)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate siegfried %s", siegfriedurl)
	}
	idx := &Indexer{
		ffProbe:    ffp,
		Siegfried:  sf,
		tempfolder: tempfolder,
	}
	return idx, nil
}

var regexpMime = regexp.MustCompile("^([^/]+)/(.+)$")

func (idx *Indexer) GetType(p []byte) (_type, subtype, mimetype string, err error) {
	f, err := ioutil.TempFile(idx.tempfolder, "indexer-")
	if err != nil {
		return "", "", "", emperror.Wrap(err, "cannot create temp file")
	}
	filename := f.Name()
	if _, err := f.Write(p); err != nil {
		f.Close()
		return "", "", "", emperror.Wrap(err, "cannot write temp file")
	}
	f.Close()
	defer os.Remove(filename)

	mediatype := http.DetectContentType(p)
	mimetype, _, err = mime.ParseMediaType(mediatype)
	if err != nil {
		return "", "", "", emperror.Wrapf(err, "cannot parse media type %s", mediatype)
	}

	var lastMime1, lastMime2 string
	var currMime1, currMime2 string

	matches := regexpMime.FindStringSubmatch(mimetype)
	if matches != nil {
		lastMime1, lastMime2 = matches[1], matches[2]
	}
	sf, err := idx.Siegfried.Identify(filename)
	if err != nil {
		return "", "", "", emperror.Wrapf(err, "cannot identiy file %s", filename)
	}

	for _, file := range sf.Files {
		for _, match := range file.Matches {
			ms := regexpMime.FindStringSubmatch(match.Mime)
			if ms != nil {
				currMime1, currMime2 = ms[1], ms[2]

				switch currMime1 {
				case "application":
					if lastMime1 == "application" && lastMime2 == "octet-stream" {
						lastMime1, lastMime2, mimetype = currMime1, currMime2, match.Mime
					}
				default:
					lastMime1, lastMime2, mimetype = currMime1, currMime2, match.Mime
				}
			}
		}
	}
	_type = lastMime1
	subtype = lastMime2
	if lastMime1 == "application" && lastMime2 == "pdf" {
		_type = "text"
		subtype = "pdf"
	}

	return
}
