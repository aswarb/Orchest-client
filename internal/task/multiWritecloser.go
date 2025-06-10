package task

import (
	"errors"
	"io"
)

type MultiWriteCloser struct {
	writers     []io.WriteCloser
	multiWriter io.Writer
}

func (m *MultiWriteCloser) Close() error {
	errs := []error{}
	for _, w := range m.writers {
		err := w.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
