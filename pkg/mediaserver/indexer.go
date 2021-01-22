package mediaserver

import (
	"github.com/goph/emperror"
	"io"
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
	mh           *MediaHandler
}

func NewIndexer(mh *MediaHandler, siegfriedurl, ffprobe, identify, convert string, identTimeout time.Duration) (*Indexer, error) {
	ffp, err := NewFFProbe(mh, ffprobe)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate ffprobe %s", ffprobe)
	}
	sf, err := NewSiegfried(mh, siegfriedurl)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate siegfried %s", siegfriedurl)
	}
	i, err := NewImagickIdentify(mh, identify, convert)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot instantiate identify %s", identify)
	}
	idx := &Indexer{
		mh:           mh,
		ffProbe:      ffp,
		Siegfried:    sf,
		identify:     i,
		identTimeout: identTimeout,
	}
	return idx, nil
}

var regexpMime = regexp.MustCompile("^([^/]+)/(.+)$")

func (idx *Indexer) SetMediaHandler(mh *MediaHandler) {
	idx.mh = mh
	idx.identify.SetMediaHandler(mh)
	idx.Siegfried.SetMediaHandler(mh)
	idx.ffProbe.SetMediaHandler(mh)
}

func (idx *Indexer) GetImageMetadata(filename string) (width, height, duration int64, mimetype, sub string, metadata map[string]interface{}, err error) {
	var result = make(map[string]interface{})
	width, height, duration, mimetype, sub, result["identify"], err = idx.identify.GetMetadata(filename, idx.identTimeout)
	metadata = result
	return
}

func (idx *Indexer) GetVideoMetadata(filename string) (width, height, duration int64, mimetype, sub string, metadata map[string]interface{}, err error) {
	var result = make(map[string]interface{})
	width, height, duration, mimetype, sub, result["ffprobe"], err = idx.ffProbe.GetMetadata(filename, idx.identTimeout)
	metadata = result
	return
}

func (idx *Indexer) GetMetadata(filename string, _type, subtype, mimetype string) (width, height, duration int64, _mimetype, sub string, metadata map[string]interface{}, err error) {
	var m string

	switch _type {
	case "image":
		width, height, duration, m, sub, metadata, err = idx.GetImageMetadata(filename)
	case "video":
		width, height, duration, m, sub, metadata, err = idx.GetVideoMetadata(filename)
	default:
		err = emperror.Wrapf(err, "invalid type %s", _type)
		return
	}
	if MimeRelevance(m) > MimeRelevance(mimetype) {
		_mimetype = m
	} else {
		_mimetype = mimetype
	}
	return
}

func (idx *Indexer) GetType(filename string) (_type, subtype, mimetype string, metadata map[string]interface{}, err error) {
	var p = make([]byte, 1024)
	f, err := os.Open(filename)
	if err != nil {
		err = emperror.Wrapf(err, "cannot open %s", filename)
		return
	}
	if _, err = io.ReadFull(f, p); err != nil {
		f.Close()
		err = emperror.Wrapf(err, "cannot read %s", filename)
		return
	}
	f.Close()

	metadata = make(map[string]interface{})
	mediatype := http.DetectContentType(p)
	mimetype, _, err = mime.ParseMediaType(mediatype)
	if err != nil {
		return "", "", "", nil, emperror.Wrapf(err, "cannot parse media type %s", mediatype)
	}
	metadata["httpdetect"] = mimetype

	var mime1, mime2 string
	matches := regexpMime.FindStringSubmatch(mimetype)
	if matches != nil {
		mime1, mime2 = matches[1], matches[2]
	}
	sf, err := idx.Siegfried.Identify(filename)
	if err != nil {
		return "", "", "", nil, emperror.Wrapf(err, "cannot identiy file %s", filename)
	}
	metadata["siegfried"] = sf

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
	_type, subtype = mime1, strings.ToLower(mime2)
	if mime1 == "application" && mime2 == "pdf" {
		_type, subtype = "text", "pdf"
	}
	return
}
