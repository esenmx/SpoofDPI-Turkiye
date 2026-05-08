package handler

import (
	"bytes"
	"errors"
	"net"
	"slices"
	"strings"
	"testing"
	"time"
)

func collect(seq func(yield func([]byte) bool)) [][]byte {
	var out [][]byte
	for c := range seq {
		out = append(out, c)
	}
	return out
}

func TestSplitInChunksEvenSplit(t *testing.T) {
	chunks := slices.Collect(splitInChunks([]byte("ABCDEFGH"), 2))
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], []byte("AB")) || !bytes.Equal(chunks[3], []byte("GH")) {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestSplitInChunksTrailingRemainder(t *testing.T) {
	chunks := slices.Collect(splitInChunks([]byte("ABCDE"), 2))
	if len(chunks) != 3 || !bytes.Equal(chunks[2], []byte("E")) {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestSplitInChunksLegacyZeroWindow(t *testing.T) {
	chunks := slices.Collect(splitInChunks([]byte("HELLO"), 0))
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], []byte("H")) || !bytes.Equal(chunks[1], []byte("ELLO")) {
		t.Fatalf("unexpected legacy split: %v", chunks)
	}
}

func TestSplitInChunksSingleByteLegacy(t *testing.T) {
	chunks := slices.Collect(splitInChunks([]byte("X"), 0))
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for single byte, got %d", len(chunks))
	}
}

func TestSplitInChunksEmpty(t *testing.T) {
	chunks := slices.Collect(splitInChunks(nil, 0))
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks for empty, got %d", len(chunks))
	}
}

// TestWriteChunksReportsErrors guards the original silent-error bug:
// writeChunks used to return (0, nil) on partial-write failures, leaving
// the caller to think the data went through and the goroutines stuck.
func TestWriteChunksReportsErrors(t *testing.T) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	dialed := make(chan *net.TCPConn, 1)
	go func() {
		c, err := net.DialTCP("tcp", nil, l.Addr().(*net.TCPAddr))
		if err != nil {
			t.Errorf("dial: %v", err)
			return
		}
		dialed <- c
	}()
	server, err := l.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := <-dialed
	defer client.Close()

	if err := client.SetWriteDeadline(time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}

	chunks := slices.Values([][]byte{bytes.Repeat([]byte{0xAB}, 64)})
	_, werr := writeChunks(client, chunks)
	if werr == nil {
		t.Fatal("expected writeChunks to surface a write error")
	}
	if !strings.Contains(werr.Error(), "chunk") {
		t.Fatalf("expected wrapped chunk error, got %v", werr)
	}
	_ = errors.Unwrap(werr)
}
