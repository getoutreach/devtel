package devtel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventWrittenToBuffer(t *testing.T) {
	var buff bytes.Buffer
	r := NewWithWriter(&buff, &testProcessor{})

	r.Record(Event{Hook: "before:deploy", Timestamp: 2147483605})

	expected := `{"key":"before:deploy","data":{"hook":"before:deploy","timestamp":2147483605}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestEventMatched(t *testing.T) {
	var buff bytes.Buffer
	r := NewWithWriter(&buff, &testProcessor{})

	r.Record(Event{Hook: "before:deploy", Timestamp: 2147483605})
	r.Record(Event{Hook: "after:deploy", Timestamp: 2147483647})

	expected := "" + // This makes it nicely arranged
		`{"key":"before:deploy","data":{"hook":"before:deploy","timestamp":2147483605}}` + "\n" +
		`{"key":"after:deploy","data":{"hook":"after:deploy","timestamp":2147483647,"duration_ms":42}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestCanRestoreFromReader(t *testing.T) {
	f := strings.NewReader("" + // This makes it nicely arranged
		`{"key":"before:deploy","data":{"hook":"before:deploy","timestamp":2147483646}}` + "\n")

	var buff bytes.Buffer
	r := NewWithWriter(&buff, &testProcessor{})

	if err := r.Restore(f); err != nil {
		t.Error(err)
	}
	r.Record(Event{Hook: "after:deploy", Timestamp: 2147483647})

	expected := `{"key":"after:deploy","data":{"hook":"after:deploy","timestamp":2147483647,"duration_ms":1}}` + "\n"
	assert.Equal(t, expected, buff.String())
}

func TestCanProcessEvents(t *testing.T) {
	var buff bytes.Buffer
	p := &testProcessor{}
	r := NewWithWriter(&buff, p)

	r.Record(Event{Hook: "before:deploy", Timestamp: 2147483605})
	r.Record(Event{Hook: "after:deploy", Timestamp: 2147483647})

	assert.NoError(t, r.Flush())
	assert.Len(t, p.lastBatch, 2)

	assert.NoError(t, r.Flush())
	assert.Len(t, p.lastBatch, 0)
}

//
