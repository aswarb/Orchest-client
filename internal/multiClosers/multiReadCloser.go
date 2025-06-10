package multiClosers

import (
	"errors"
	"io"
)

type MultiReadCloser struct {
	readers []io.ReadCloser
	io.Reader
}

func (m *MultiReadCloser) Close() error {
	errs := []error{}
	for _, w := range m.readers {
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

func MakeMultiReadCloser(readClosers ...io.ReadCloser) *MultiReadCloser {
	readers := []io.Reader{}
	for _, rc := range readClosers {
		readers = append(readers, rc)
	}

	mr := io.MultiReader(readers...)
	mrc := MultiReadCloser{
		readers: readClosers,
		Reader:  mr,
	}

	return &mrc
}
