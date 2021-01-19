package indexer

import (
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

type SFIdentifier struct {
	Name    string `json:"name,omitempty"`
	Details string `json:"details,omitempty"`
}

type SFMatches struct {
	Ns      string `json:"ns,omitempty"`
	Id      string `json:"id,omitempty"`
	Format  string `json:"format,omitempty"`
	Version string `json:"version,omitempty"`
	Mime    string `json:"mime,omitempty"`
	Basis   string `json:"basis,omitempty"`
	Warning string `json:"warning,omitempty"`
}

type SFFiles struct {
	Filename string      `json:"filename,omitempty"`
	Filesize int64       `json:"filesize,omitempty"`
	Modified string      `json:"modified,omitempty"`
	Errors   string      `json:"errors,omitempty"`
	Matches  []SFMatches `json:"matches,omitempty"`
}

type SF struct {
	Siegfried   string         `json:"siegfried,omitempty"`
	Scandate    string         `json:"scandate,omitempty"`
	Signature   string         `json:"signature,omitempty"`
	Created     string         `json:"created,omitempty"`
	Identifiers []SFIdentifier `json:"identfiers,omitempty"`
	Files       []SFFiles      `json:"files,omitempty"`
}

type Siegfried struct {
	url string
}

func NewSiegfried(urlstring string) (*Siegfried, error) {
	sf := &Siegfried{
		url: urlstring,
	}
	return sf, nil
}

func (sf *Siegfried) Identify(filename string) (*SF, error) {
	urlstring := strings.Replace(sf.url, "[[PATH]]", strings.Replace(url.QueryEscape(filepath.ToSlash(filename)), "+", "%20", -1), -1)
	resp, err := http.Get(urlstring)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot query siegfried - %v", urlstring)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, emperror.Wrapf(err, "error reading body - %v", urlstring)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok - %v -> %v: %s", urlstring, resp.Status, string(bodyBytes))
	}

	result := SF{}
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, emperror.Wrapf(err, "error decoding json - %v", string(bodyBytes))
	}
	if len(result.Files) == 0 {
		return nil, emperror.Wrapf(err, "no file in sf result - %v", string(bodyBytes))
	}

	return &result, nil
}
