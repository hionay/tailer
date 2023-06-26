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

const DefaultWaitDuration = 1 * time.Second
const DefaultDashString = "━"

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

type afterReader struct {
	rd       io.Reader
	tm       *time.Timer
	stopch   chan struct{}
	resetch  chan struct{}
	dur      time.Duration
	stopOnce sync.Once
}

func newAfterReader(rd io.Reader, dur time.Duration) *afterReader {
	tm := time.NewTimer(dur)
	tm.Stop()
	return &afterReader{
		rd:      rd,
		tm:      tm,
		stopch:  make(chan struct{}),
		resetch: make(chan struct{}),
		dur:     dur,
	}
}

func (ar *afterReader) Read(p []byte) (int, error) {
	n, err := ar.rd.Read(p)
	if err != nil {
		ar.stopOnce.Do(func() {
			close(ar.stopch)
		})
		return n, err
	}
	select {
	case ar.resetch <- struct{}{}:
	default:
	}
	return n, nil
}

func (ar *afterReader) tailReceiver() <-chan struct{} {
	tailch := make(chan struct{})
	go func() {
		defer close(tailch)
		defer ar.tm.Stop()
		for {
			select {
			case <-ar.stopch:
				return
			case <-ar.resetch:
				if !ar.tm.Stop() {
					select {
					case <-ar.tm.C:
					default:
					}
				}
				ar.tm.Reset(ar.dur)
			case <-ar.tm.C:
				tailch <- struct{}{}
			}
		}
	}()
	return tailch
}
