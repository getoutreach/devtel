package store

import "fmt"

type Cursor struct {
	currIndex int

	items []map[string]interface{}
}

func NewCursor(items []map[string]interface{}) *Cursor {
	return &Cursor{
		currIndex: -1,
		items:     items,
	}
}

func (c *Cursor) Next() bool {
	if c.currIndex+1 < len(c.items) {
		c.currIndex++
		return true
	}
	return false
}

func (c *Cursor) Value(v IndexMarshaller) error {
	if c.currIndex < 0 {
		return fmt.Errorf("cursor is not positioned")
	}

	return v.UnmarshalRecord(c.items[c.currIndex])
}

func (c *Cursor) Len() int {
	return len(c.items)
}
