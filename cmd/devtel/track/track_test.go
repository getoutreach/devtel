package track_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/getoutreach/gobox/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"

	"github.com/getoutreach/devtel/cmd/devtel/track"
)

func TestTrackEvent(t *testing.T) {
	log.SetOutput(io.Discard)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "devtel", r.Header.Get("X-OUTREACH-CLIENT-APP-ID"))
		assert.Equal(t, "testKey", r.Header.Get("X-OUTREACH-CLIENT-LOGGING"))

		defer r.Body.Close()

		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		assert.NoError(t, err)

		assert.Contains(t, string(b), `"hook":"before:build"`)
		assert.Contains(t, string(b), `"execution_id":"031cb474-c2f4-433f-863e-684c35c8d5ac"`)
		assert.Contains(t, string(b), `"status":"info"`)
		assert.Contains(t, string(b), `line":"devspace deploy [flags]"`)
		assert.Contains(t, string(b), `name":"deploy"`)
		assert.Contains(t, string(b), `"email":"yoda@outreach.io"`)

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	os.Setenv("OUTREACH_TELEFORK_ENDPOINT", server.URL)
	os.Setenv("DEV_EMAIL", "yoda@outreach.io")

	os.Setenv("DEVSPACE_PLUGIN_EVENT", "before:build")
	os.Setenv("DEVSPACE_PLUGIN_EXECUTION_ID", "031cb474-c2f4-433f-863e-684c35c8d5ac")
	os.Setenv("DEVSPACE_PLUGIN_COMMAND", "deploy")
	os.Setenv("DEVSPACE_PLUGIN_COMMAND_LINE", "devspace deploy [flags]")
	os.Setenv("DEVSPACE_PLUGIN_COMMAND_FLAGS", `["--namespace","yoda--bento1a","--no-warn"]`)
	os.Setenv("DEVSPACE_PLUGIN_COMMAND_ARGS", "")
	os.Setenv("DEVSPACE_PLUGIN_ERROR", "")

	os.Args = []string{"devtel", "track"}
	app := &cli.App{
		Name: "devtel",
		Commands: []*cli.Command{
			track.NewCommand("testKey"),
		},
	}

	app.Run(os.Args)
}
