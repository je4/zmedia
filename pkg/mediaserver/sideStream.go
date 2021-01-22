package mediaserver

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/goph/emperror"
	"hash"
	"io/ioutil"
	"os"
)

type SideStream struct {
	buffer     bytes.Buffer
	size       int
	pos        int
	checksum   string
	sha256Hash hash.Hash
	tempfolder string
	tempsize   int64
	f          *os.File
	tempfile   string
}

func NewSideStream(tempfolder string, size int) (*SideStream, error) {
	ss := &SideStream{
		size:       size,
		buffer:     bytes.Buffer{},
		sha256Hash: sha256.New(),
		tempfolder: tempfolder,
	}
	return ss, nil
}

func (ss *SideStream) Open() (string, error) {
	var err error
	ss.f, err = ioutil.TempFile(ss.tempfolder, "indexer-")
	if err != nil {
		return "", emperror.Wrap(err, "cannot create temp file")
	}
	ss.tempfile = ss.f.Name()
	return ss.tempfile, nil
}

func (ss *SideStream) Close() {
	if ss.f != nil {
		ss.f.Close()
		ss.f = nil
	}
}

func (ss *SideStream) Clear() {
	if ss.f != nil {
		ss.f.Close()
	}
	if ss.tempfile != "" {
		os.Remove(ss.tempfile)
		ss.tempfile = ""
	}
}

func (ss *SideStream) Write(p []byte) (n int, err error) {
	if _, err := ss.sha256Hash.Write(p); err != nil {
		return 0, emperror.Wrap(err, "error in hash write")
	}

	if ss.size > 0 && ss.size <= ss.pos {
		return len(p), nil
	}
	l := ss.size - ss.pos
	if len(p) >= l {
		if ss.f != nil {
			if _, err := ss.f.Write(p); err != nil {
				return 0, emperror.Wrapf(err, "cannot write tempfile %s", ss.tempfile)
			}
		} else {
			n, err = ss.buffer.Write(p[0:l])
			if err != nil {
				return 0, emperror.Wrapf(err, "cannot write %v bytes", l)
			}
		}
		ss.pos += n
	} else {
		n, err := ss.buffer.Write(p)
		if err != nil {
			return 0, emperror.Wrapf(err, "cannot write %v bytes", l)
		}
		ss.pos += n
	}
	return len(p), nil
}

func (ss *SideStream) GetSHA256() string {
	return fmt.Sprintf("%x", ss.sha256Hash.Sum(nil))
}

func (ss *SideStream) GetBytes() []byte {
	return ss.buffer.Bytes()
}
