package track

import (
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
			if err := t.Init(c.Context); err != nil {
				panic(err)
			}

			defer t.Flush(c.Context)

			event := devspace.EventFromEnv()
			t.Track(c.Context, event)

			return nil
		},
	}
}
