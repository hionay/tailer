package tailer

import (
	"io"
	"os"
	"time"
)

const DefaultWaitDuration = 1 * time.Second
const DefaultDashString = "─"

type options struct {
	inrd          io.Reader
	outwr         io.Writer
	dashString    string
	afterDuration time.Duration
	noColor       bool
}

func getDefaultOptions() options {
	return options{
		inrd:          os.Stdin,
		outwr:         os.Stdout,
		dashString:    DefaultDashString,
		afterDuration: DefaultWaitDuration,
		noColor:       false,
	}
}

type TailerOptionFunc func(*options)

func WithInputReader(rd io.Reader) TailerOptionFunc {
	return func(opts *options) {
		opts.inrd = rd
	}
}

func WithOutputWriter(wr io.Writer) TailerOptionFunc {
	return func(opts *options) {
		opts.outwr = wr
	}
}

func WithNoColor(noColor bool) TailerOptionFunc {
	return func(opts *options) {
		opts.noColor = noColor
	}
}

func WithDashString(str string) TailerOptionFunc {
	return func(opts *options) {
		opts.dashString = str
	}
}

func WithAfterDuration(dur time.Duration) TailerOptionFunc {
	return func(opts *options) {
		opts.afterDuration = dur
	}
}
