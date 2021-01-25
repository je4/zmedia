package filesystem

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"
)

// errSeeker is returned by ServeContent's sizeFunc when the content
// doesn't seek properly. The underlying Seeker's error text isn't
// included in the sizeFunc reply so it's not sent over HTTP to end
// users.
var errSeeker = errors.New("seeker can't seek")

// errNoOverlap is returned by serveContent's parseRange if first-byte-pos of
// all of the byte-range-spec values is greater than the content size.
var errNoOverlap = errors.New("invalid range: failed to overlap")

type NotFoundError struct {
	err error
}

func (nf *NotFoundError) Error() string {
	return fmt.Sprintf("file not found: %v", nf.err)
}

func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// PutObjectOptions represents options specified by user for PutObject call
type FilePutOptions struct {
	Progress    io.Reader
	ContentType string
}

// GetObjectOptions represents options specified by user for GetObject call
type FileGetOptions struct {
	VersionID string
}

type FileStatOptions struct {
}

type FolderCreateOptions struct {
	ObjectLocking bool
}

type ReadSeekerCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type FileSystem interface {
	BucketExists(folder string) (bool, error)
	BucketCreate(folder string, opts FolderCreateOptions) error
	GETUrl(folder, name string, valid time.Duration) (*url.URL, error)
	FileExists(folder, name string) (bool, error)
	FileGet(folder, name string, opts FileGetOptions) ([]byte, error)
	FilePut(folder, name string, data []byte, opts FilePutOptions) error
	FileWrite(folder, name string, r io.Reader, size int64, opts FilePutOptions) error
	FileRead(folder, name string, w io.Writer, size int64, opts FileGetOptions) error
	FileOpenRead(folder, name string, opts FileGetOptions) (ReadSeekerCloser, os.FileInfo, error)
	FileStat(folder, name string, opts FileStatOptions) (os.FileInfo, error)
	String() string
	Protocol() string
	IsLocal() bool
}
