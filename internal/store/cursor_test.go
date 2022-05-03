// Copyright 2022 Outreach Corporation. All Rights Reserved.

package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCursorEmpty(t *testing.T) {
	c := Cursor{}
	assert.False(t, c.Next())
}

func TestCursorNext(t *testing.T) {
	c := NewCursor([]map[string]interface{}{
		{"id": "before:a"},
	})
	assert.True(t, c.Next())

	var e payload
	assert.NoError(t, c.Value(&e))
	assert.Equal(t, "before:a", e.ID)
}
