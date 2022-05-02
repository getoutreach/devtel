package store

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type payload struct {
	ID        string `json:"id"`
	Content   string `json:"content,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func (p *payload) Key() string {
	return p.ID
}
func (p *payload) MarshalRecord(addField func(name string, value interface{})) {
	addField("id", p.ID)
	if p.Content != "" {
		addField("content", p.Content)
	}
}

func (p *payload) UnmarshalRecord(data map[string]interface{}) error {
	if id, ok := data["id"]; ok && id.(string) != "" {
		p.ID = id.(string)
	} else {
		return fmt.Errorf("payload requires ID (%v)", data)
	}

	if content, ok := data["content"]; ok {
		p.Content = content.(string)
	}
	if timestamp, ok := data["timestamp"]; ok {
		switch t := timestamp.(type) {
		case float64:
			p.Timestamp = int64(t)
		case int64:
			p.Timestamp = t
		}
	}

	return nil
}

func openAppender(buff *TestClosableBuffer) func(path string) (io.WriteCloser, error) {
	return func(path string) (io.WriteCloser, error) {
		return buff, nil
	}
}

func TestStoreAppend(t *testing.T) {
	var buff TestClosableBuffer
	s := New(&Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id2"}))

	expected := "" +
		`{"key":"id1","data":{"id":"id1"}}` + "\n" +
		`{"key":"id2","data":{"id":"id2"}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestStoreGet(t *testing.T) {
	var buff TestClosableBuffer
	s := New(&Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id2"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1", Content: "content1"}))

	var val payload
	s.Get(context.Background(), "id1", &val)
	assert.NotEmpty(t, val)
	assert.Equal(t, "content1", val.Content)
}

func TestStoreGetAll(t *testing.T) {
	var buff TestClosableBuffer
	s := New(&Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id2"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1", Content: "content1"}))

	cursor := s.GetAll(context.Background())
	assert.Equal(t, 2, cursor.Len())

	var curr payload
	assert.True(t, cursor.Next())
	assert.NoError(t, cursor.Value(&curr))
	assert.Equal(t, "id2", curr.ID)
	assert.NotEqual(t, "content1", curr.Content)

	assert.True(t, cursor.Next())
	assert.NoError(t, cursor.Value(&curr))
	assert.Equal(t, "id1", curr.ID)
	assert.Equal(t, "content1", curr.Content)
}

func TestStoreGetUnprocessed(t *testing.T) {
	var buff TestClosableBuffer
	s := New(&Options{
		OpenAppend: openAppender(&buff),
	})

	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id2"}))
	assert.Nil(t, s.Append(context.Background(), &payload{ID: "id1", Content: "content1"}))

	cursor := s.GetAll(context.Background())
	assert.Equal(t, 2, cursor.Len())

	var curr payload
	assert.True(t, cursor.Next())
	assert.NoError(t, cursor.Value(&curr))
	assert.Equal(t, "id2", curr.ID)
	assert.NotEqual(t, "content1", curr.Content)

	assert.True(t, cursor.Next())
	assert.NoError(t, cursor.Value(&curr))
	assert.Equal(t, "id1", curr.ID)
	assert.Equal(t, "content1", curr.Content)

	toProcess := []IndexMarshaller{
		&payload{ID: "id1", Content: "content1"},
	}
	assert.NoError(t, s.MarkProcessed(context.Background(), toProcess))
	cursor = s.GetUnprocessed(context.Background())

	assert.Equal(t, cursor.Len(), 1)
}

func TestAddDefaultField(t *testing.T) {
	var buff TestClosableBuffer
	s := New(&Options{
		OpenAppend: openAppender(&buff),
	})

	s.AddDefaultField("dev.email", "yoda@outreach.io")

	assert.NoError(t, s.Append(context.Background(), &payload{ID: "id1"}))
	expected := `{"key":"id1","data":{"dev":{"email":"yoda@outreach.io"},"id":"id1"}}` + "\n"

	assert.Equal(t, expected, buff.String())
}
