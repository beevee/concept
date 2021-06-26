package main

import (
	"log"
	"os"

	"github.com/beevee/concept/internal"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "concept",
		Usage: "run various upkeep tasks via Notion API",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "token",
				Usage:    "Notion integration token",
				EnvVars:  []string{"CONCEPT_TOKEN"},
				Required: true,
			},
		},
		Commands: []*cli.Command{
			&internal.TrimCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
