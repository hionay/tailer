package tailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

type Tailer struct {
	pw         io.WriteCloser
	aftrd      *afterReader
	stopch     chan struct{}
	opts       options
	mu         sync.Mutex
	started    bool
	isTerminal bool
}

func New(opts ...TailerOptionFunc) *Tailer {
	tl := &Tailer{
		stopch: make(chan struct{}),
		opts:   getDefaultOptions(),
	}
	for _, opt := range opts {
		opt(&tl.opts)
	}
	return tl
}

func (tl *Tailer) Run(ctx context.Context) error {
	tl.mu.Lock()
	if tl.started {
		tl.mu.Unlock()
		return errors.New("already started")
	}
	tl.started = true
	tl.mu.Unlock()

	defer close(tl.stopch)

	if f, ok := tl.opts.outwr.(*os.File); ok && f != nil {
		tl.isTerminal = term.IsTerminal(int(f.Fd()))
	}
	pr, pw := io.Pipe()
	tl.pw = pw
	aftrd := newAfterReader(tl.opts.inrd, tl.opts.afterDuration)
	tl.aftrd = aftrd

	go tl.worker(ctx)
	go func() {
		_, _ = io.Copy(pw, aftrd)
	}()
	_, err := io.Copy(tl.opts.outwr, pr)
	return err
}

func (tl *Tailer) Close() error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if tl.started {
		tl.started = false
		return tl.pw.Close()
	}
	return nil
}

func (tl *Tailer) worker(ctx context.Context) {
	lastWriteTm := time.Now()
	tailch := tl.aftrd.tailReceiver()
	for {
		select {
		case <-ctx.Done():
			_ = tl.pw.Close()
			return
		case <-tl.stopch:
			_ = tl.pw.Close()
			return
		case <-tailch:
			tmpassed := time.Since(lastWriteTm).Truncate(100 * time.Millisecond)
			tl.writeTailer(tmpassed)
			lastWriteTm = time.Now()
		}
	}
}

func (tl *Tailer) writeTailer(dur time.Duration) {
	now := time.Now()
	datestr := now.Format("2006-01-02")
	tmstr := now.Format("15:04:05")
	durstr := dur.String()
	filled := len(datestr) + len(tmstr) + len(durstr) + 3
	if !tl.opts.noColor {
		datestr = color.GreenString(datestr)
		tmstr = color.YellowString(tmstr)
		durstr = color.BlueString(durstr)
	}

	var width int
	if tl.isTerminal {
		if f, ok := tl.opts.outwr.(*os.File); ok && f != nil {
			var err error
			width, _, err = term.GetSize(int(f.Fd()))
			if err != nil {
				width = 80
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(datestr + " " + tmstr + " " + durstr + " ")
	if rpt := width - filled; rpt > 0 {
		sb.WriteString(strings.Repeat(tl.opts.dashString, rpt))
	}
	_, _ = fmt.Fprintln(tl.opts.outwr, sb.String())
}
