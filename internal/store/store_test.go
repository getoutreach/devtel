package store

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type payload struct {
	ID      string `json:"id"`
	Content string `json:"content,omitempty"`
}

func payloadID(data interface{}) string {
	return data.(payload).ID
}

func restorePayload(m map[string]interface{}) interface{} {
	return payload{
		ID:      m["id"].(string),
		Content: m["content"].(string),
	}
}

func openAppender(buff *TestClosableBuffer) func(path string) (io.WriteCloser, error) {
	return func(path string) (io.WriteCloser, error) {
		return buff, nil
	}
}

func TestStoreAppend(t *testing.T) {
	var buff TestClosableBuffer
	s := New(payloadID, &Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(payload{ID: "id1"}))
	assert.Nil(t, s.Append(payload{ID: "id2"}))

	expected := "" +
		`{"key":"id1","data":{"id":"id1"}}` + "\n" +
		`{"key":"id2","data":{"id":"id2"}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestStoreGet(t *testing.T) {
	var buff TestClosableBuffer
	s := New(payloadID, &Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(payload{ID: "id1"}))
	assert.Nil(t, s.Append(payload{ID: "id2"}))
	assert.Nil(t, s.Append(payload{ID: "id1", Content: "content1"}))

	val := s.Get("id1")
	assert.NotNil(t, val)
	assert.Equal(t, "content1", val.(payload).Content)
}

func TestStoreGetAll(t *testing.T) {
	var buff TestClosableBuffer
	s := New(payloadID, &Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(payload{ID: "id1"}))
	assert.Nil(t, s.Append(payload{ID: "id2"}))
	assert.Nil(t, s.Append(payload{ID: "id1", Content: "content1"}))

	val := s.GetAll()
	assert.NotNil(t, val)
	assert.Equal(t, 2, len(val))
	assert.Equal(t, "id2", val[0].(payload).ID)
	assert.Equal(t, "content1", val[1].(payload).Content)
}

func TestStoreGetUnprocessed(t *testing.T) {
	var buff TestClosableBuffer
	s := New(payloadID, &Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(payload{ID: "id1"}))
	assert.Nil(t, s.Append(payload{ID: "id2"}))
	assert.Nil(t, s.Append(payload{ID: "id1", Content: "content1"}))

	val := s.GetAll()
	assert.NotNil(t, val)
	assert.Len(t, val, 2)
	assert.Equal(t, "id2", val[0].(payload).ID)
	assert.Equal(t, "content1", val[1].(payload).Content)

	toProcess := []interface{}{
		payload{ID: "id1", Content: "content1"},
	}
	assert.NoError(t, s.MarkProcessed(toProcess))
	val = s.GetUnprocessed()

	assert.Len(t, val, 1)
}

func TestRestore(t *testing.T) {
	var buff TestClosableBuffer
	s := New(payloadID, &Options{
		OpenAppend:       openAppender(&buff),
		RestoreConverter: restorePayload,
	})

	r := strings.NewReader("" +
		`{"key":"id1","data":{"id":"id1","content":"content1"}}` + "\n" +
		`{"key":"id1","data":{"id":"id1","content":"content42"}}` + "\n" +
		`{"key":"id2","data":{"id":"id2","content":"content2"}}` + "\n")

	s.Restore(r)

	val := s.Get("id1")
	assert.NotNil(t, val)
	assert.Equal(t, "content42", val.(payload).Content)

	val = s.Get("id3")
	assert.Nil(t, val)
}
