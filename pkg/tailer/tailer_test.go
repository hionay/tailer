package tailer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTailerRunNoInput(t *testing.T) {
	in := make([]byte, 0)
	var out bytes.Buffer
	tl := New(
		WithInputReader(bytes.NewReader(in)),
		WithOutputWriter(&out),
		WithAfterDuration(0),
	)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := tl.Run(context.Background())
		assert.NoError(t, err)
	}()
	time.AfterFunc(10*time.Millisecond, func() { _ = tl.Close() })
	wg.Wait()
	require.Equal(t, 0, out.Len())
}

func TestTailerRunNoInputWithDuration(t *testing.T) {
	in := make([]byte, 0)
	var out bytes.Buffer
	tl := New(
		WithInputReader(bytes.NewReader(in)),
		WithOutputWriter(&out),
		WithAfterDuration(10*time.Millisecond),
	)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := tl.Run(context.Background())
		assert.NoError(t, err)
	}()
	time.AfterFunc(30*time.Millisecond, func() { _ = tl.Close() })
	wg.Wait()
	require.Equal(t, 0, out.Len())
}

func TestTailerRun(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	var out bytes.Buffer
	tl := New(
		WithInputReader(pr),
		WithOutputWriter(&out),
		WithAfterDuration(5*time.Millisecond),
	)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := tl.Run(context.Background())
		assert.NoError(t, err)
	}()
	txt := "hello world"
	fmt.Fprintln(pw, txt)
	time.AfterFunc(50*time.Millisecond, func() { _ = tl.Close() })
	wg.Wait()
	_ = pw.Close()
	require.True(t, strings.HasPrefix(out.String(), txt+"\n"))
	require.True(t, strings.HasSuffix(out.String(), DefaultDashString+"\n"))
}
