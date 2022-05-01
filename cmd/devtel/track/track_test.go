package track_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/urfave/cli/v2"

	"github.com/getoutreach/devtel/cmd/devtel/track"
	"github.com/stretchr/testify/assert"
)

func TestTrackEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "devtel", r.Header.Get("X-OUTREACH-CLIENT-APP-ID"))
		assert.Equal(t, "testKey", r.Header.Get("X-OUTREACH-CLIENT-LOGGING"))

		defer r.Body.Close()

		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		//nolint:lll // Why: Comparing actual value, the string is long.
		assert.Regexp(t, `^[{"event_name":"devspace_hook_event","hook":"before:build","execution_id":"031cb474-c2f4-433f-863e-684c35c8d5ac","status":"info","command":{"name":"deploy","line":"devspace deploy [flags]","flags":["--namespace","yoda--bento1a","--no-warn"]},"timestamp":\d+}]$`, string(b))

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	os.Setenv("OUTREACH_TELEFORK_ENDPOINT", server.URL)

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
			track.NewCmd("testKey"),
		},
	}

	app.Run(os.Args)
}
