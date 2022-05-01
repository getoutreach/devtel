package track

import (
	"encoding/json"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/getoutreach/devtel/internal/devspace"
	"github.com/getoutreach/devtel/internal/telefork"
)

func NewCmd(teleforkAPIKey string) *cli.Command {
	return &cli.Command{
		Name:  "track",
		Usage: "Track events",
		Action: func(c *cli.Context) error {
			t := devspace.NewTracker(telefork.NewProcessor(c.App.Name, teleforkAPIKey))
			if err := t.Init(); err != nil {
				panic(err)
			}

			defer t.Flush()

			event := devspace.EventFromEnv()

			//nolint
			json.NewEncoder(os.Stdout).Encode(event)

			t.Track(event)

			return nil
		},
	}
}
