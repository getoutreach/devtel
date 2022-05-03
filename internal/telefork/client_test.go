// Copyright 2022 Outreach Corporation. All Rights Reserved.

package telefork

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/getoutreach/devtel/internal/devspace"
	"github.com/stretchr/testify/assert"
)

func TestClientSendsEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "testApp", r.Header.Get("X-OUTREACH-CLIENT-APP-ID"))
		assert.Equal(t, "testKey", r.Header.Get("X-OUTREACH-CLIENT-LOGGING"))

		defer r.Body.Close()

		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		assert.Equal(t, `[{"hook":"before:deploy","timestamp":2147483605,"@timestamp":"0001-01-01T00:00:00Z"}]`, string(b))

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	os.Setenv("OUTREACH_TELEFORK_ENDPOINT", server.URL)
	client := NewClientWithHTTPClient("testApp", "testKey", server.Client())
	err := client.SendEvents(context.Background(), []interface{}{
		devspace.Event{Hook: "before:deploy", Timestamp: 2147483605, TimestampTag: time.Time{}},
	})
	assert.NoError(t, err)
}
