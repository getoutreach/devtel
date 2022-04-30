package store

import "bytes"

type TestClosableBuffer struct {
	bytes.Buffer
}

func (TestClosableBuffer) Close() error {
	return nil
}
