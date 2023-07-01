package tailer

import (
	"context"
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
	readch     chan struct{}
	opts       options
	wg         sync.WaitGroup
	mu         sync.Mutex
	isTerminal bool
}

func New(opts ...TailerOptionFunc) *Tailer {
	tl := &Tailer{
		readch: make(chan struct{}, 1),
		opts:   getDefaultOptions(),
	}
	for _, opt := range opts {
		opt(&tl.opts)
	}
	return tl
}

func (tl *Tailer) Run(ctx context.Context) error {
	pr, pw := io.Pipe()
	tl.mu.Lock()
	tl.pw = pw
	tl.mu.Unlock()

	if f, ok := tl.opts.outwr.(*os.File); ok && f != nil {
		tl.isTerminal = term.IsTerminal(int(f.Fd()))
	}

	tl.wg.Add(1)
	go tl.worker(ctx)
	go func() {
		_, _ = io.Copy(pw, tl.opts.inrd)
		_ = tl.Close()
	}()

	writerFn := writeFunc(func(p []byte) (int, error) {
		tl.readch <- struct{}{}
		tl.mu.Lock()
		defer tl.mu.Unlock()
		return tl.opts.outwr.Write(p)
	})
	_, err := io.Copy(writerFn, pr)
	close(tl.readch)
	tl.wg.Wait()
	return err
}

func (tl *Tailer) Close() error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.pw.Close()
}

func (tl *Tailer) worker(ctx context.Context) {
	defer tl.wg.Done()

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			_ = tl.Close()
			return
		case _, ok := <-tl.readch:
			if !ok {
				_ = tl.Close()
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(tl.opts.afterDuration)
		case ts := <-timer.C:
			tl.printLine(ts, last)
			last = ts
		}
	}
}

func (tl *Tailer) printLine(ts, last time.Time) {
	var (
		datestr  = ts.Format("2006-01-02")
		tmstr    = ts.Format("15:04:05")
		sincestr = ts.Sub(last).Truncate(100 * time.Millisecond).String()
	)
	filled := len(datestr) + len(tmstr) + len(sincestr) + 3
	if !tl.opts.noColor {
		datestr = color.GreenString(datestr)
		tmstr = color.YellowString(tmstr)
		sincestr = color.BlueString(sincestr)
	}

	width := 80
	if tl.isTerminal {
		if f, ok := tl.opts.outwr.(*os.File); ok && f != nil {
			w, _, err := term.GetSize(int(f.Fd()))
			if err == nil {
				width = w
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(datestr + " " + tmstr + " " + sincestr + " ")
	if count := width - filled; count > 0 {
		sb.WriteString(strings.Repeat(tl.opts.dashString, count))
	}
	tl.mu.Lock()
	_, _ = fmt.Fprintln(tl.opts.outwr, sb.String())
	tl.mu.Unlock()
}

type writeFunc func(p []byte) (int, error)

func (wf writeFunc) Write(p []byte) (int, error) { return wf(p) }
