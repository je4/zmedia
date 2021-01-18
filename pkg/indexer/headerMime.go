package indexer

import (
	"bytes"
	"github.com/goph/emperror"
	"net/http"
)

type HeaderMime struct {
	buffer bytes.Buffer
	size   int
	pos    int
}

func (hm *HeaderMime) Write(p []byte) (n int, err error) {
	if hm.size <= hm.pos {
		return len(p), nil
	}
	l := hm.size - hm.pos
	if len(p) >= l {
		n, err := hm.buffer.Write(p[0:l])
		if err != nil {
			return 0, emperror.Wrapf(err, "cannot write %v bytes", l)
		}
		hm.pos += n
	} else {
		n, err := hm.buffer.Write(p)
		if err != nil {
			return 0, emperror.Wrapf(err, "cannot write %v bytes", l)
		}
		hm.pos += n
	}
	return len(p), nil
}

func (hm *HeaderMime) GetMime() string {
	return http.DetectContentType(hm.buffer.Bytes())
}
