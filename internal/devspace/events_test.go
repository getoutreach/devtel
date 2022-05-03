// Copyright 2022 Outreach Corporation. All Rights Reserved.

package devspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStartEvent(t *testing.T) {
	assert.Equal(t, "", getBeforeHook("before:deploy"))
	assert.Equal(t, "before:deploy", getBeforeHook("after:deploy"))
	assert.Equal(t, "before:deploy:app", getBeforeHook("after:deploy:app"))
	assert.Equal(t, "before:deploy:metrics", getBeforeHook("after:deploy:metrics"))
	assert.Equal(t, "buildCommand:before:execute", getBeforeHook("buildCommand:interrupt"))
}
