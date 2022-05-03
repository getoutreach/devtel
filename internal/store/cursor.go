// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains cursor implementation for accessing generic event data.

package store

import "fmt"

// Cursor implements iteration over generic event data.
type Cursor struct {
	currIndex int

	items []map[string]interface{}
}

// NewCursor creates new cursor instance for given items.
func NewCursor(items []map[string]interface{}) *Cursor {
	return &Cursor{
		currIndex: -1,
		items:     items,
	}
}

// Next moves cursor to next item.
func (c *Cursor) Next() bool {
	if c.currIndex+1 < len(c.items) {
		c.currIndex++
		return true
	}
	return false
}

// Value sets v to current value. Returns error if value cannot be unmarshaled.
func (c *Cursor) Value(v IndexMarshaller) error {
	if c.currIndex < 0 {
		return fmt.Errorf("cursor is not positioned")
	}

	return v.UnmarshalRecord(c.items[c.currIndex])
}

// Len returns number of items in cursor.
func (c *Cursor) Len() int {
	return len(c.items)
}
