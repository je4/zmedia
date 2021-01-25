package database

import (
	"fmt"
	"github.com/goph/emperror"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"net/url"
	"regexp"
	"strings"
)

type Storage struct {
	Fs           filesystem.FileSystem `json:"-"`
	db           *MediaDatabase        `json:"-"`
	FSType       string                `json:"fstype"`
	Id           int64                 `json:"id"`
	Name         string                `json:"name"`
	Filebase     string                `json:"filebase"`
	DataDir      string                `json:"data"`
	VideoDir     string                `json:"video"`
	SubmasterDir string                `json:"submasterdir"`
	TempDir      string                `json:"temp"`
	JWTKey       string                `json:"jwtkey"`
}

func NewStorage(mdb *MediaDatabase, id int64, name, filebase, datadir, videodir, submasterdir, tempdir, jwtkey string) (*Storage, error) {
	if datadir == "" {
		datadir = "data"
	}
	if videodir == "" {
		videodir = "video"
	}
	if tempdir == "" {
		tempdir = "temp"
	}

	urlbase, err := url.Parse(filebase)
	if err != nil {
		return nil, emperror.Wrapf(err, "invalid URL for storage [%v] %s - %s", id, name, filebase)
	}
	/*
		if strings.ToLower(urlbase.Scheme) != "fs" {
			return nil, fmt.Errorf("invalid scheme for filesystem %s", filebase)
		}
	*/
	fsname := strings.ToLower(fmt.Sprintf("%s://%s", urlbase.Scheme, urlbase.Host))
	fs, ok := mdb.fss[fsname]
	if !ok {
		return nil, fmt.Errorf("unknown filesystem %s", filebase)
	}

	stor := &Storage{
		db:           mdb,
		Fs:           fs,
		Id:           id,
		FSType:       fs.String(),
		Name:         name,
		Filebase:     filebase,
		DataDir:      datadir,
		VideoDir:     videodir,
		SubmasterDir: submasterdir,
		TempDir:      tempdir,
		JWTKey:       jwtkey,
	}
	return stor, nil
}

var regexpBaseBucket = regexp.MustCompile(`://[^/]+/([^/]+)$`)

func (s Storage) GetBucket() (string, error) {
	matches := regexpBaseBucket.FindStringSubmatch(s.Filebase)
	if matches == nil {
		return "", fmt.Errorf("no bucket in filebase %s", s.Filebase)
	}
	return matches[1], nil
}

func (s Storage) Store() error {
	return nil
}
