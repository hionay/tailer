package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
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

const (
	commandExec      = "exec"
	commandExecShort = "e"
)

var version string

func main() {
	app := &cli.App{
		Name:    "tailer",
		Version: version,
		Usage:   "a simple CLI tool to insert lines when command output stops",
		Before: func(c *cli.Context) error {
			if c.IsSet(flagDash) {
				if c.String(flagDash) == "" {
					return errors.New("dash char cannot be empty")
				}
				if len(c.String(flagDash)) > 1 {
					return errors.New("dash char cannot be longer than 1 character")
				}
			}
			return nil
		},
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
				Usage:   "dash character to print",
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
		Commands: []*cli.Command{
			{
				Name:  commandExec,
				Usage: "Execute a command and tail its output",
				Before: func(c *cli.Context) error {
					if c.NArg() == 0 {
						return errors.New("arguments cannot be empty")
					}
					return nil
				},
				Aliases: []string{commandExecShort},
				Action:  execAction,
			},
		},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

func execAction(c *cli.Context) error {
	first, tail := parseCommand(c.Args().First())
	if tail == nil {
		tail = c.Args().Tail()
	}
	cmd := exec.CommandContext(c.Context, first, tail...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	opts := []tailer.TailerOptionFunc{
		tailer.WithAfterDuration(c.Duration(flagAfter)),
		tailer.WithDashString(c.String(flagDash)),
		tailer.WithInputReader(pr),
	}
	if c.Bool(flagNoColor) {
		opts = append(opts, tailer.WithNoColor(true))
	}
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("failed to run command: %v", err)
		}
		pw.Close()
	}()
	tl := tailer.New(opts...)
	return tl.Run(c.Context)
}

func parseCommand(args string) (string, []string) {
	if strings.Contains(args, " ") {
		split := strings.Split(args, " ")
		return split[0], split[1:]
	}
	return args, nil
}
