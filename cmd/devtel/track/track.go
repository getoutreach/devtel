package track

import (
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/getoutreach/devtel/internal/devspace"
	"github.com/getoutreach/devtel/internal/telefork"
	"github.com/getoutreach/gobox/pkg/trace"
)

// commonProps is a helper function to get the  common properties for telefork telemetry
func commonProps() map[string]interface{} {
	commonProps := map[string]interface{}{
		"os.name": runtime.GOOS,
		"os.arch": runtime.GOARCH,
	}

	var email string
	if os.Getenv("DEV_EMAIL") != "" {
		email = os.Getenv("DEV_EMAIL")
	} else if b, err := exec.Command("git", "config", "user.email").Output(); err == nil {
		email = strings.TrimSuffix(string(b), "\n")
	}

	if email != "" && strings.HasSuffix(email, "@outreach.io") {
		commonProps["dev.email"] = email
		if u, err := user.Current(); err == nil {
			commonProps["os.user"] = u.Username
		}
		if hostname, err := os.Hostname(); err == nil {
			commonProps["os.hostname"] = hostname
		}
		path, err := os.Getwd()
		if err == nil {
			commonProps["os.workDir"] = path
		}
	}

	return commonProps
}

func NewCmd(teleforkAPIKey string) *cli.Command {
	return &cli.Command{
		Name:  "track",
		Usage: "Track events",
		Action: func(c *cli.Context) error {
			ctx := trace.StartCall(c.Context, "store.Append")
			defer trace.EndCall(ctx)

			t := devspace.NewTracker(telefork.NewProcessor(c.App.Name, teleforkAPIKey))
			if err := t.Init(ctx); err != nil {
				//nolint:errcheck // Why: We don't wat to crash devspace because of telemetry errors.
				trace.SetCallStatus(ctx, err)
				return nil
			}
			p := commonProps()
			for k, v := range p {
				t.AddDefaultField(k, v)
			}

			defer t.Flush(c.Context)

			event := devspace.EventFromEnv()
			t.Track(c.Context, event)

			return nil
		},
	}
}
