package sse

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// NewEOLSplitterFunc returns a bufio.SplitFunc tied to a new EOLSplitter instance.
func NewEOLSplitterFunc() bufio.SplitFunc {
	splitter := NewEOLSplitter()
	return splitter.Split
}

// EOLSplitter is the custom split function to handle CR LF, CR, and LF as end-of-line.
type EOLSplitter struct {
	prevCR bool
}

// NewEOLSplitter creates a new EOLSplitter instance.
func NewEOLSplitter() *EOLSplitter {
	return &EOLSplitter{prevCR: false}
}

const crlfLen = 2

// Split function to handle CR LF, CR, and LF as end-of-line.
func (s *EOLSplitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Check if the previous data ended with a CR
	if s.prevCR {
		s.prevCR = false
		if len(data) > 0 && data[0] == '\n' {
			return 1, nil, nil // Skip the LF following the previous CR
		}
	}

	// Search for the first occurrence of CR LF, CR, or LF
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' {
			if i+1 < len(data) && data[i+1] == '\n' {
				// Found CR LF
				return i + crlfLen, data[:i], nil
			}
			// Found CR
			if !atEOF && i == len(data)-1 {
				// If CR is the last byte, and not EOF, then need to check if
				// the next byte is LF.
				//
				// save the state and request more data
				s.prevCR = true
				return 0, nil, nil
			}
			return i + 1, data[:i], nil
		}
		if data[i] == '\n' {
			// Found LF
			return i + 1, data[:i], nil
		}
	}

	// If at EOF, we have a final, non-terminated line. Return it.
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

type ServerSentEvent struct {
	ID      string // ID of the event
	Data    string // Data of the event
	Event   string // Type of the event
	Retry   int    // Retry time in milliseconds
	Comment string // Comment
}

// GJSON
func (e *ServerSentEvent) GJSON(path string) gjson.Result {
	return gjson.Get(e.Data, path)
}

type Scanner struct {
	readCloser io.ReadCloser

	scanner     *bufio.Scanner
	next        ServerSentEvent
	err         error
	readComment bool
}

func NewScanner(r io.Reader, readComment bool) *Scanner {
	s := &Scanner{
		readComment: readComment,
	}

	s.setReader(r)

	return s
}

// setReader
func (s *Scanner) setReader(r io.Reader) {
	// N.B. The bufio.ScanLines handles `\r?\n``, but not `\r` itself as EOL, as
	// the SSE spec requires
	//
	// See: https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream
	//
	// scanner.Split(bufio.ScanLines)

	var readCloser io.ReadCloser
	if rc, ok := r.(io.ReadCloser); ok {
		readCloser = rc
	} else {
		readCloser = io.NopCloser(r)
	}

	s.readCloser = readCloser

	scanner := bufio.NewScanner(r)
	scanner.Split(NewEOLSplitterFunc())
	s.scanner = scanner
}

// Tee
func (s *Scanner) Tee(w io.Writer) {
	type readCloser struct {
		io.Reader
		io.Closer
	}

	s.readCloser = &readCloser{
		Reader: io.TeeReader(s.readCloser, w),
		Closer: s.readCloser,
	}

	s.setReader(s.readCloser)
}

func (s *Scanner) Close() error {
	return s.readCloser.Close()
}

func (s *Scanner) Next() bool {
	// Zero the next event before scanning a new one
	var event ServerSentEvent
	s.next = event

	var dataLines []string

	var seenNonEmptyLine bool

	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())

		if line == "" {
			if seenNonEmptyLine {
				break
			}

			continue
		}

		seenNonEmptyLine = true
		switch {
		case strings.HasPrefix(line, "id: "):
			event.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "data: "):
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		case strings.HasPrefix(line, "event: "):
			event.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "retry: "):
			retry, err := strconv.Atoi(strings.TrimPrefix(line, "retry: "))
			if err == nil {
				event.Retry = retry
			}
			// ignore invalid retry values
		case strings.HasPrefix(line, ":"):
			if s.readComment {
				event.Comment = strings.TrimPrefix(line, ":")
			}
			// ignore comment line
		default:
			// ignore unknown lines
		}
	}

	s.err = s.scanner.Err()

	if !seenNonEmptyLine {
		return false
	}

	event.Data = strings.Join(dataLines, "\n")
	s.next = event

	return true
}

func (s *Scanner) Event() ServerSentEvent {
	return s.next
}

func (s *Scanner) Err() error {
	return s.err
}
