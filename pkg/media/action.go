package media

import (
	"fmt"
	"io"
)

type Action interface {
	Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error)
	Close()
	GetType() string
}

type GenericAction struct{}

func (ga *GenericAction) Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error) {
	if action != "master" {
		return nil, fmt.Errorf("generic action %s not possible", action)
	}
	return nil, fmt.Errorf("generic action not implemented")
}

func (ga *GenericAction) Close() {
}

func (ga *GenericAction) GetType() string {
}
