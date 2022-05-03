package telefork

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/getoutreach/devtel/internal/devspace"
	"github.com/stretchr/testify/assert"
)

func TestTeleforkProcessor(t *testing.T) {
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

	tp := &Processor{
		client: client,
	}

	tp.ProcessRecords(context.Background(), []interface{}{
		devspace.Event{Hook: "before:deploy", Timestamp: 2147483605},
	})
}
