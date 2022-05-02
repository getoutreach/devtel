package store

import (
	"bytes"
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
	}).(*store)
	s.logDir = ""

	assert.NoError(t, s.Init())

	var beforeDeploy, afterDeploy, afterBuild payload
	assert.NoError(t, s.Get("before:deploy", &beforeDeploy))
	assert.NoError(t, s.Get("after:deploy", &afterDeploy))
	assert.NoError(t, s.Get("after:build", &afterBuild))

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

	assert.NoError(t, s.Init())

	s.Append(&payload{ID: "before:deploy", Content: "content1"})
	s.Append(&payload{ID: "after:deploy", Content: "content1"})

	var beforeDeploy, afterDeploy, afterBuild payload
	s.Get("before:deploy", &beforeDeploy)
	s.Get("after:deploy", &afterDeploy)
	s.Get("after:build", &afterBuild)

	assert.NotEmpty(t, beforeDeploy)
	assert.NotNil(t, afterDeploy)
	assert.Empty(t, afterBuild)
}
