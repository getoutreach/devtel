package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValuesAreOverwritten(t *testing.T) {
	m := make(map[string]interface{})

	adder := addField(m)

	adder("a", 1)
	adder("a.b", 2)
	adder("a.b.c", 3)

	assert.Equal(t, m, map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 3,
			},
		},
	})
}

func TestValuesAreAdded(t *testing.T) {
	m := make(map[string]interface{})

	adder := addField(m)

	adder("a", 1)
	adder("b.a", 2)
	adder("b.b", 3)

	assert.Equal(t, m, map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{
			"a": 2,
			"b": 3,
		},
	})
}
