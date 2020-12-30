package media

import "io"

type Action interface {
	Do(meta *CoreMeta, action string, params []string, reader io.Reader, writer io.Writer) (*CoreMeta, error)
	Close()
}
