package track

import (
	"encoding/json"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/getoutreach/devtel/internal/devspace"
	"github.com/getoutreach/devtel/internal/telefork"
)

func NewCmd(appName, teleforkAPIKey string) *cli.Command {
	return &cli.Command{
		Name:  "track",
		Usage: "Track events",
		Action: func(c *cli.Context) error {
			log.Println("Starting devtel track")

			t := devspace.NewTracker(telefork.NewProcessor(appName, teleforkAPIKey))
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
