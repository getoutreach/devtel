package store_test

import (
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

func id(val interface{}) string {
	return val.(testEvent).ID
}
func restore(val map[string]interface{}) interface{} {
	return testEvent{
		ID: val["id"].(string),
	}
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
	s := store.New(id, &store.Options{
		LogDir:           tmpDir,
		RestoreConverter: restore,
	})
	assert.NoError(t, s.Init())
	assert.NoError(t, s.Append(testEvent{ID: eventID}))
}

func restoreFormFile(t *testing.T) {
	s := store.New(id, &store.Options{
		LogDir:           tmpDir,
		RestoreConverter: restore,
	})
	assert.NoError(t, s.Init())

	assert.NotNil(t, s.Get(eventID))
}

func processEvents(t *testing.T) {
	s := store.New(id, &store.Options{
		LogDir:           tmpDir,
		RestoreConverter: restore,
	})
	assert.NoError(t, s.Init())
	id1 := s.Get(eventID)
	assert.NotNil(t, id1)

	s.MarkProcessed([]interface{}{id1})
}
