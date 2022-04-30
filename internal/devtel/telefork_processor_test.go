package devtel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/getoutreach/devtel/internal/telefork"
	"github.com/stretchr/testify/assert"
)

func TestTeleforkProcessor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "testApp", r.Header.Get("X-OUTREACH-CLIENT-APP-ID"))
		assert.Equal(t, "testKey", r.Header.Get("X-OUTREACH-CLIENT-LOGGING"))

		defer r.Body.Close()

		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		assert.Equal(t, `[{"hook":"before:deploy","timestamp":2147483605}]`, string(b))

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	os.Setenv("OUTREACH_TELEFORK_ENDPOINT", server.URL)
	client := telefork.NewClientWithHTTPClient("testApp", "testKey", server.Client())

	tp := &teleforkProcessor{
		client: client,
	}

	tp.ProcessRecords([]interface{}{
		Event{Hook: "before:deploy", Timestamp: 2147483605},
	})
}
