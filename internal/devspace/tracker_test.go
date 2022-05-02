package devspace

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"testing/fstest"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/stretchr/testify/assert"
)

var beforeEvent = `{
	"event": "devspace_hook",
	"hook": "before:deploy",
	"execution_id": "9714f00a-b998-49e7-97a9-a8e2051905f7",
	"status": "info",
	"command": {
		"name": "deploy",
		"line": "devspace deploy [flags]",
		"flags": [
			"--config",
			"/Users/yoda/outreach/force/devspace.yaml",
			"--namespace",
			"force--bento1a",
			"--no-warn",
			"true"
		]
	},
	"timestamp": 1651388142703
}`

var afterEvent = `{
	"event": "devspace_hook",
	"hook": "after:deploy",
	"execution_id": "9714f00a-b998-49e7-97a9-a8e2051905f7",
	"status": "info",
	"command": {
		"name": "deploy",
		"line": "devspace deploy [flags]",
		"flags": [
			"--config",
			"/Users/yoda/outreach/force/devspace.yaml",
			"--namespace",
			"force--bento1a",
			"--no-warn",
			"true"
		]
	},
	"timestamp": 1651388151749
}`

// The beforeEvent and afterEvent are real events from a test run.
// For the sake of understanding what's insinde, the JSON is indented.
// Here, we remove the indentation so we can use them in the tests as expected.
// Unmarshal and marshal back, but without indentation.
//nolint:gochecknoinits // Why: helps understand the test data
func init() {
	normalize := func(orig string) string {
		var event Event
		if err := json.Unmarshal([]byte(orig), &event); err != nil {
			panic(err)
		}

		if b, err := json.Marshal(event); err != nil {
			panic(err)
		} else {
			return string(b)
		}
	}

	beforeEvent = normalize(beforeEvent)
	afterEvent = normalize(afterEvent)
}

func TestEventWrittenToBuffer(t *testing.T) {
	var buff store.TestClosableBuffer
	s := store.New(&store.Options{
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(&testProcessor{}, WithStore(s))

	var before Event
	assert.NoError(t, json.Unmarshal([]byte(beforeEvent), &before))
	r.Track(context.Background(), &before)

	var b Event
	assert.NoError(t, s.Get("9714f00a-b998-49e7-97a9-a8e2051905f7_before:deploy", &b))
	assert.Equal(t, before, b)
}

func TestEventMatched(t *testing.T) {
	var buff store.TestClosableBuffer
	s := store.New(&store.Options{
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(&testProcessor{}, WithStore(s))

	var before, after Event
	assert.NoError(t, json.Unmarshal([]byte(beforeEvent), &before))
	assert.NoError(t, json.Unmarshal([]byte(afterEvent), &after))

	r.Track(context.Background(), &before)
	r.Track(context.Background(), &after)

	assert.NoError(t, s.Get("9714f00a-b998-49e7-97a9-a8e2051905f7_after:deploy", &after))
	assert.Equal(t, int64(9046), after.Duration)
}

func TestCanUseRestoredEvents(t *testing.T) {
	logFS := make(fstest.MapFS)
	logFS["1.log"] = &fstest.MapFile{
		Data: []byte(`{"key":"9714f00a-b998-49e7-97a9-a8e2051905f7_before:deploy","data":` + beforeEvent + `}` + "\n"),
	}
	var buff store.TestClosableBuffer
	p := &testProcessor{}
	s := store.New(&store.Options{
		LogFS: logFS,
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(p, WithStore(s))
	assert.NoError(t, r.Init(context.Background()))

	var after Event
	assert.NoError(t, json.Unmarshal([]byte(afterEvent), &after))
	r.Track(context.Background(), &after)

	assert.NoError(t, s.Get("9714f00a-b998-49e7-97a9-a8e2051905f7_after:deploy", &after))
	assert.Equal(t, int64(9046), after.Duration)
}

func TestCanProcessEvents(t *testing.T) {
	var buff store.TestClosableBuffer
	p := &testProcessor{}
	s := store.New(&store.Options{
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(p, WithStore(s))

	var before, after Event
	assert.NoError(t, json.Unmarshal([]byte(beforeEvent), &before))
	assert.NoError(t, json.Unmarshal([]byte(afterEvent), &after))

	r.Track(context.Background(), &before)
	r.Track(context.Background(), &after)

	assert.NoError(t, r.Flush(context.Background()))
	assert.Len(t, p.lastBatch, 2)

	assert.NoError(t, r.Flush(context.Background()))
	assert.Len(t, p.lastBatch, 0)
}
