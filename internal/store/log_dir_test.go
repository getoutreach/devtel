// Copyright 2022 Outreach Corporation. All Rights Reserved.

package store

import (
	"bytes"
	"context"
	"io"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

type testWriteCloser struct {
	fs   fstest.MapFS
	path string
	buff bytes.Buffer
}

func (w *testWriteCloser) Write(p []byte) (n int, err error) {
	return w.buff.Write(p)
}

func (w *testWriteCloser) Close() error {
	if f, ok := w.fs[w.path]; ok {
		f.Data = w.buff.Bytes()
	} else {
		w.fs[w.path] = &fstest.MapFile{
			Data: w.buff.Bytes(),
		}
	}

	w.buff.Reset()

	return nil
}

func TestInit(t *testing.T) {
	logFS := make(fstest.MapFS)
	logFS["test1.txt"] = &fstest.MapFile{
		Data: []byte(`{"key":"before:deploy","data":{"id":"before:deploy","timestamp":2147483605}}` + "\n"),
	}
	logFS["test2.txt"] = &fstest.MapFile{
		Data: []byte(`{"key":"after:deploy","data":{"id":"after:deploy","timestamp":2147483647}}` + "\n"),
	}

	w := testWriteCloser{
		path: "latest.txt",
		fs:   logFS,
	}

	s := New(&Options{
		LogDir: "./",
		LogFS:  logFS,
		OpenAppend: func(path string) (io.WriteCloser, error) {
			return &w, nil
		},
	})
	s.logDir = ""

	assert.NoError(t, s.Init(context.Background()))

	var beforeDeploy, afterDeploy, afterBuild payload
	assert.NoError(t, s.Get(context.Background(), "before:deploy", &beforeDeploy))
	assert.NoError(t, s.Get(context.Background(), "after:deploy", &afterDeploy))
	assert.NoError(t, s.Get(context.Background(), "after:build", &afterBuild))

	assert.NotEmpty(t, beforeDeploy)
	assert.NotNil(t, afterDeploy)
	assert.Empty(t, afterBuild)
}

func TestAppendOpener(t *testing.T) {
	logFS := make(fstest.MapFS)
	w := testWriteCloser{
		path: "latest.txt",
		fs:   logFS,
	}
	s := New(&Options{
		LogFS: logFS,
		OpenAppend: func(path string) (io.WriteCloser, error) {
			return &w, nil
		},
	})

	assert.NoError(t, s.Init(context.Background()))

	s.Append(context.Background(), &payload{ID: "before:deploy", Content: "content1"})
	s.Append(context.Background(), &payload{ID: "after:deploy", Content: "content1"})

	var beforeDeploy, afterDeploy, afterBuild payload
	s.Get(context.Background(), "before:deploy", &beforeDeploy)
	s.Get(context.Background(), "after:deploy", &afterDeploy)
	s.Get(context.Background(), "after:build", &afterBuild)

	assert.NotEmpty(t, beforeDeploy)
	assert.NotNil(t, afterDeploy)
	assert.Empty(t, afterBuild)
}
