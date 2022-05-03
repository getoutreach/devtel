package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/stretchr/testify/assert"
)

var tmpDir = "../../tmp/test"
var eventID = fmt.Sprintf("id_%d", time.Now().UnixNano())

type testEvent struct {
	ID string `json:"id"`
}

func (e testEvent) Key() string {
	return e.ID
}
func (e *testEvent) MarshalRecord(addField func(name string, value interface{})) {
	addField("id", e.ID)
}

func (e *testEvent) UnmarshalRecord(data map[string]interface{}) error {
	e.ID = data["id"].(string)

	return nil
}

func TestStoreData(t *testing.T) {
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0o777)

	// This file will get picked up automatically by the store
	f, err := os.CreateTemp(tmpDir, "*.log")
	assert.NoError(t, err)

	tempFile := f.Name()
	assert.NoError(t, f.Close())

	appendToFile(t)
	expected := fmt.Sprintf(`{"key":%q,"data":{"id":%q}}`, eventID, eventID) + "\n"

	b, err := os.ReadFile(tempFile)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(b))

	restoreFormFile(t)

	b, err = os.ReadFile(tempFile)
	assert.NoError(t, err)
	// The file doesn't change, it's just being read
	assert.Equal(t, expected, string(b))

	processEvents(t)

	b, err = os.ReadFile(tempFile)
	assert.NoError(t, err)

	expected += fmt.Sprintf(`{"key":%q,"data":{"id":%q},"processed":true}`, eventID, eventID) + "\n"

	assert.Equal(t, expected, string(b))
}

func appendToFile(t *testing.T) {
	s := store.New(&store.Options{
		LogDir: tmpDir,
	})
	assert.NoError(t, s.Init(context.Background()))
	assert.NoError(t, s.Append(context.Background(), &testEvent{ID: eventID}))
}

func restoreFormFile(t *testing.T) {
	s := store.New(&store.Options{
		LogDir: tmpDir,
	})
	assert.NoError(t, s.Init(context.Background()))
	var e testEvent
	s.Get(context.Background(), eventID, &e)
	assert.NotEmpty(t, e)
}

func processEvents(t *testing.T) {
	s := store.New(&store.Options{
		LogDir: tmpDir,
	})
	assert.NoError(t, s.Init(context.Background()))
	var id1 testEvent
	s.Get(context.Background(), eventID, &id1)
	assert.NotNil(t, id1)

	s.MarkProcessed(context.Background(), []store.IndexMarshaller{&id1})
}
