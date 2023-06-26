package tailer

import (
	"io"
	"sync"
	"time"
)

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
