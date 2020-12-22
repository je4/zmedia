package database

import (
	"fmt"
	"github.com/goph/emperror"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"net/url"
	"strings"
)

type Storage struct {
	fs           filesystem.FileSystem `json:"-"`
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
	if strings.ToLower(urlbase.Scheme) != "fs" {
		return nil, fmt.Errorf("invalid scheme for filesystem %s", filebase)
	}
	fs, ok := mdb.fss[strings.ToLower(urlbase.Host)]
	if !ok {
		return nil, fmt.Errorf("unknown filesystem %s", filebase)
	}

	stor := &Storage{
		db:           mdb,
		fs:           fs,
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
