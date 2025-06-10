package task

import (
	"errors"
	"io"
)

type MultiWriteCloser struct {
	writers []io.WriteCloser
	io.Writer
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

func MakeMultiWriteCloser(writeClosers ...io.WriteCloser) *MultiWriteCloser {
	writers := []io.Writer{}
	for _, wc := range writeClosers {
		writers = append(writers, wc)
	}

	mw := io.MultiWriter(writers...)
	mwc := MultiWriteCloser{
		writers: writeClosers,
		Writer:  mw,
	}

	return &mwc
}
