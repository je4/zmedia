package mediaserver

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/goph/emperror"
	"hash"
)

type SideStream struct {
	buffer     bytes.Buffer
	size       int
	pos        int
	checksum   string
	sha256Hash hash.Hash
}

func NewSideStream(size int) (*SideStream, error) {
	ss := &SideStream{
		size:       size,
		buffer:     bytes.Buffer{},
		sha256Hash: sha256.New(),
	}
	return ss, nil
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
		n, err := ss.buffer.Write(p[0:l])
		if err != nil {
			return 0, emperror.Wrapf(err, "cannot write %v bytes", l)
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
