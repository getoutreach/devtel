package devspace

import (
	"encoding/json"
	"io"
	"testing"
	"testing/fstest"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/stretchr/testify/assert"
)

var beforeEvent = `{
	"event_name: "devspace_hook_event",
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
			"true",
			"--var",
			"[IMAGE_REGISTRY=us-docker.pkg.dev/outreach-docker/outreach-devenv/plisy-devenv]"
		]
	},
	"timestamp": 1651388142703
}`

var afterEvent = `{
	"event_name: "devspace_hook_event",
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
			"true",
			"--var",
			"[IMAGE_REGISTRY=us-docker.pkg.dev/outreach-docker/outreach-devenv/plisy-devenv]"
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
	var m Event
	if err := json.Unmarshal([]byte(beforeEvent), &m); err != nil {
		panic(err)
	}
	if b, err := json.Marshal(m); err != nil {
		panic(err)
	} else {
		beforeEvent = string(b)
	}

	if err := json.Unmarshal([]byte(afterEvent), &m); err != nil {
		panic(err)
	}
	if b, err := json.Marshal(m); err != nil {
		panic(err)
	} else {
		afterEvent = string(b)
	}
}

func TestEventWrittenToBuffer(t *testing.T) {
	var buff store.TestClosableBuffer
	s := store.New(eventKey, &store.Options{
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(&testProcessor{}, WithStore(s))

	var before Event
	assert.NoError(t, json.Unmarshal([]byte(beforeEvent), &before))
	r.Track(&before)

	expected := `{"key":"9714f00a-b998-49e7-97a9-a8e2051905f7_before:deploy","data":` + beforeEvent + `}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestEventMatched(t *testing.T) {
	var buff store.TestClosableBuffer
	s := store.New(eventKey, &store.Options{
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(&testProcessor{}, WithStore(s))

	var before, after Event
	assert.NoError(t, json.Unmarshal([]byte(beforeEvent), &before))
	assert.NoError(t, json.Unmarshal([]byte(afterEvent), &after))

	r.Track(&before)
	r.Track(&after)

	expected := "" + // This makes it nicely arranged
		`{"key":"9714f00a-b998-49e7-97a9-a8e2051905f7_before:deploy","data":` + beforeEvent + `}` + "\n" +
		`{"key":"9714f00a-b998-49e7-97a9-a8e2051905f7_after:deploy","data":` + afterEvent[:len(afterEvent)-1] + `,"duration_ms":9046}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestCanRestoreOnInit(t *testing.T) {
	logFS := make(fstest.MapFS)
	logFS["1.log"] = &fstest.MapFile{
		Data: []byte(`{"key":"9714f00a-b998-49e7-97a9-a8e2051905f7_before:deploy","data":` + beforeEvent + `}` + "\n"),
	}
	var buff store.TestClosableBuffer
	p := &testProcessor{}
	s := store.New(eventKey, &store.Options{
		LogFS: logFS,

		RestoreConverter: eventFromMap,
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(p, WithStore(s))

	if err := r.Init(); err != nil {
		t.Error(err)
	}
	var after Event
	assert.NoError(t, json.Unmarshal([]byte(afterEvent), &after))
	r.Track(&after)

	expected := `{"key":"9714f00a-b998-49e7-97a9-a8e2051905f7_after:deploy","data":` +
		afterEvent[:len(afterEvent)-1] + `,"duration_ms":9046}}` + "\n"
	assert.Equal(t, expected, buff.String())
}

func TestCanProcessEvents(t *testing.T) {
	var buff store.TestClosableBuffer
	p := &testProcessor{}
	s := store.New(eventKey, &store.Options{
		RestoreConverter: eventFromMap,
		OpenAppend: func(key string) (io.WriteCloser, error) {
			return &buff, nil
		},
	})
	r := NewTracker(p, WithStore(s))

	var before, after Event
	assert.NoError(t, json.Unmarshal([]byte(beforeEvent), &before))
	assert.NoError(t, json.Unmarshal([]byte(afterEvent), &after))

	r.Track(&before)
	r.Track(&after)

	assert.NoError(t, r.Flush())
	assert.Len(t, p.lastBatch, 2)

	assert.NoError(t, r.Flush())
	assert.Len(t, p.lastBatch, 0)
}
