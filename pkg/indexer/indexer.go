package indexer

import (
	"github.com/goph/emperror"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

/*
holistic function to give some mimetypes a relevance
*/
func MimeRelevance(mimetype string) (relevance int) {
	if mimetype == "" {
		return 0
	}
	if mimetype == "application/octet-stream" {
		return 1
	}
	if mimetype == "text/plain" {
		return 2
	}
	if mimetype == "audio/mpeg" {
		return 2
	}
	if strings.HasPrefix(mimetype, "application/") {
		return 3
	}
	if strings.HasPrefix(mimetype, "text/") {
		return 4
	}
	return 100
}

type Indexer struct {
	ffProbe      *FFProbe
	Siegfried    *Siegfried
	identify     *ImagickIdentify
	identTimeout time.Duration
	tempfolder   string
}

func NewIndexer(siegfriedurl, ffprobe, identify string, identTimeout time.Duration, tempfolder string) (*Indexer, error) {
	ffp, err := NewFFProbe(ffprobe)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate ffprobe %s", ffprobe)
	}
	sf, err := NewSiegfried(siegfriedurl)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate siegfried %s", siegfriedurl)
	}
	i, err := NewImagickIdentify(identify)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate identify %s", identify)
	}
	idx := &Indexer{
		ffProbe:      ffp,
		Siegfried:    sf,
		identify:     i,
		identTimeout: identTimeout,
		tempfolder:   tempfolder,
	}
	return idx, nil
}

var regexpMime = regexp.MustCompile("^([^/]+)/(.+)$")

func (idx *Indexer) GetImageMetadata(urlstring string) (width, height, duration int64, mimetype, sub string, metadata interface{}, err error) {
	return idx.identify.GetMetadata(urlstring, idx.identTimeout)
}

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

	var mime1, mime2 string

	matches := regexpMime.FindStringSubmatch(mimetype)
	if matches != nil {
		mime1, mime2 = matches[1], matches[2]
	}
	sf, err := idx.Siegfried.Identify(filename)
	if err != nil {
		return "", "", "", emperror.Wrapf(err, "cannot identiy file %s", filename)
	}

	rel := MimeRelevance(mimetype)
	for _, file := range sf.Files {
		for _, match := range file.Matches {
			currrel := MimeRelevance(match.Mime)
			if currrel > rel {
				ms := regexpMime.FindStringSubmatch(match.Mime)
				if ms != nil {
					mime1, mime2, mimetype = ms[1], ms[2], match.Mime
					rel = MimeRelevance(mimetype)
				}
			}
		}
	}
	_type, subtype = mime1, mime2
	if mime1 == "application" && mime2 == "pdf" {
		_type, subtype = "text", "pdf"
	}

	return
}

func (idx *Indexer) GetMetadata(urlstring string, _type, subtype, mimetype string) (width, height, duration int64, _mimetype, sub string, metadata interface{}, err error) {
	var m string
	switch _type {
	case "image":
		width, height, duration, m, sub, metadata, err = idx.GetImageMetadata(urlstring)
	default:
		err = emperror.Wrapf(err, "invalid type %s", _type)
	}
	if MimeRelevance(m) > MimeRelevance(mimetype) {
		_mimetype = m
	} else {
		_mimetype = mimetype
	}
	return
}
