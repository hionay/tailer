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
	go io.Copy(pw, tl.opts.inrd) //nolint:errcheck

	var err error
	buf := make([]byte, 2048)
	for {
		var n int
		n, err = pr.Read(buf)
		if err != nil {
			close(tl.readch)
			break
		}
		tl.mu.Lock()
		_, _ = tl.opts.outwr.Write(buf[:n])
		tl.mu.Unlock()
		tl.readch <- struct{}{}
	}
	tl.wg.Wait()
	if err == io.EOF {
		return nil
	}
	return err
}

func (tl *Tailer) Close() error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.pw.Close()
}

func (tl *Tailer) worker(ctx context.Context) {
	defer tl.wg.Done()

	tm := time.NewTimer(tl.opts.afterDuration)
	if !tm.Stop() {
		<-tm.C
	}
	lastWriteTm := time.Now()
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
			if !tm.Stop() {
				select {
				case <-tm.C:
				default:
				}
			}
			tm.Reset(tl.opts.afterDuration)
		case <-tm.C:
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
	sb.WriteString(datestr + " " + tmstr + " " + durstr + " ")
	if rpt := width - filled; rpt > 0 {
		sb.WriteString(strings.Repeat(tl.opts.dashString, rpt))
	}
	tl.mu.Lock()
	_, _ = fmt.Fprintln(tl.opts.outwr, sb.String())
	tl.mu.Unlock()
}
