package textstore

import "io"

type TextStore interface {
	Open(id string) (io.ReadCloser, error)
	Create(id string) (io.WriteCloser, error)
	Delete(id string) error
	ForEach(prefix string, f func(id string) error) error
}
