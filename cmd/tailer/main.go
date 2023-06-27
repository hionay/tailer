package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/hionay/tailer/pkg/tailer"
)

const (
	flagAfter      = "after"
	flagAfterShort = "a"
	flagDash       = "dash"
	flagDashShort  = "d"
	flagNoColor    = "no-color"
)

func main() {
	app := &cli.App{
		Name:  "tailer",
		Usage: "a simple CLI tool to insert lines when command output stops",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  flagNoColor,
				Usage: "disable color output",
			},
			&cli.DurationFlag{
				Name:    flagAfter,
				Usage:   "duration to wait after last output",
				Value:   tailer.DefaultWaitDuration,
				Aliases: []string{flagAfterShort},
			},
			&cli.StringFlag{
				Name:    flagDash,
				Usage:   "dash string to print",
				Value:   tailer.DefaultDashString,
				Aliases: []string{flagDashShort},
			},
		},
		Action: func(c *cli.Context) error {
			opts := []tailer.TailerOptionFunc{
				tailer.WithAfterDuration(c.Duration(flagAfter)),
				tailer.WithDashString(c.String(flagDash)),
			}
			if c.Bool(flagNoColor) {
				opts = append(opts, tailer.WithNoColor(true))
			}
			tl := tailer.New(opts...)
			return tl.Run(c.Context)
		},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
