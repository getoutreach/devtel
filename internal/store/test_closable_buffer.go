// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This package contains the implementation of mock io.Closer

package store

import "bytes"

// TestClosableBuffer wraps a bytes.Buffer and implements io.Closer.
type TestClosableBuffer struct {
	bytes.Buffer
}

// Close implements io.Closer. Does nothing.
func (TestClosableBuffer) Close() error {
	return nil
}
