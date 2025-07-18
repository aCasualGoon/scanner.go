// Package scanner provides utilities for scanning text (strings).
package scanner

import (
	"strings"
	"unicode/utf8"
)

// EOF represents the synthetic termination rune of a string.
const EOF rune = -1

// A TextPosition represents a position within a piece of text (string).
type TextPosition struct {
	// Idx is the offset in bytes from the beginning of the string.
	Idx  int
	// Line is the line component of the position. Can also be seen as the number of line breaks since the beginning of the string plus one.
	Line int
	// Column is the column component of the position. Can also be seen as the number of runes since the last line break plus one.
	Col  int
}

// A RuneSpan represents a rune within text, including the matching positional data.
type RuneSpan struct {
	// Rune is the rune.
	Rune rune
	// Pos is the position the rune is at.
	Pos  TextPosition
	// End is the position one after the rune.
	End  TextPosition
}

// A Scanner holds the data needed to scan a piece of text.
type Scanner struct {
	TextPosition
	text string

	markedPos          TextPosition
	isComplexSinceMark bool  // true if can't be directly sliced
}

// NewScanner creates a new scanner for the given piece of text initialized to the TextPosition at index 0.
func NewScanner(text string) *Scanner {
	return NewScannerAt(text, TextPosition { Idx: 0, Line: 1, Col: 1 })
}

// NewScannerAt creates a new scanner for the given piece of text initialized to the given starting TextPosition.
func NewScannerAt(text string, startingPosition TextPosition) *Scanner {
  return &Scanner{
		TextPosition:       startingPosition,
		text:               text,
		markedPos:          startingPosition,
		isComplexSinceMark: false,
	}
}

// IsEOF returns whether the scanner has moved past the end of the input.
func (scan *Scanner) IsEOF() bool {
  return scan.Idx >= len(scan.text)
}

// Pop returns the rune at the current scanner position and advances the position to the next rune.
// If the current position is past the end of the text, ScanEOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scan *Scanner) Pop() rune {
	if scan.IsEOF() {
		return EOF
	}

	r, w := utf8.DecodeRuneInString(scan.text[scan.Idx:])

	scan.Idx += w
	scan.Col++

	switch r {
	case '\n':
		scan.Line++
		scan.Col = 1

	case '\r':
		scan.Line++
		scan.Col = 1

		scan.isComplexSinceMark = true

		// check if part of CRLF. if so, skip LF too.
		if !scan.IsEOF() {
			if nextR, nextW := utf8.DecodeRuneInString(scan.text[scan.Idx:]); nextR == '\n' {
				scan.Idx += nextW
			}
		}

		// normalize CR and CRLF to LF
		return '\n'

	case '\\':
		savedPos := scan.TextPosition
		// using pop() automatically handles line break normalization and EOF guard
		if scan.Pop() != '\n' {
			// we need to reset the scanner position if it's just a regular backslash
			scan.TextPosition = savedPos
			break
		}

		scan.isComplexSinceMark = true

		// just return whatever the rune after the escaped line break is
		return scan.Pop()
	}

	return r
}

// PopSpan works like Scanner.Pop but returns the corresponding RuneSpan instead of just the plain rune.
func (scan *Scanner) PopSpan() RuneSpan {
	startPos := scan.TextPosition
	r := scan.Pop()
	return RuneSpan{
		Rune: r,
		Pos:  startPos,
		End:  scan.TextPosition,
	}
}

// Peek returns the rune at the current scanner position without advancing.
// If the current position is past the end of the text, ScanEOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scan *Scanner) Peek() rune {
	savedPos := scan.TextPosition
	r := scan.Pop()
	scan.TextPosition = savedPos
	return r
}

// PeekSpan works like Scanner.Peek but returns the corresponding RuneSpan instead of just the plain rune.
func (scan *Scanner) PeekSpan() RuneSpan {
	span := scan.PopSpan()
	scan.TextPosition = span.Pos
	return span
}

// Next consumes the rune at the current scanner position and returns the next rune.
// If the current position is past the end of the text, ScanEOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scan *Scanner) Next() rune {
	scan.Pop()
	return scan.Peek()
}

// NextSpan works like Scanner.Next but returns the corresponding RuneSpan instead of just the plain rune.
func (scan *Scanner) NextSpan() RuneSpan {
	scan.Pop()
	return scan.PeekSpan()
}

// Mark marks the rune at the current scanner position to be the first rune in the next Scanner.Slice or Scanner.SliceIncl call.
func (scan *Scanner) Mark() {
	scan.markedPos = scan.TextPosition
	scan.isComplexSinceMark = false
}

// Marked returns the TextPosition that was last marked using Scanner.Mark
func (scan *Scanner) Marked() TextPosition {
  return scan.markedPos
}

// Slice returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (exclusive).
func (scan *Scanner) Slice() string {
  if scan.markedPos.Idx >= len(scan.text) {
    return ""
  }

	slice := scan.text[scan.markedPos.Idx:scan.Idx]

	if scan.isComplexSinceMark {
		slice = strings.ReplaceAll(slice, "\r\n", "\n")
		slice = strings.ReplaceAll(slice, "\r", "\n")
		slice = strings.ReplaceAll(slice, "\\\n", "")
	}

	return slice
}

// SliceIncl returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (inclusive).
func (scan *Scanner) SliceIncl() string {
  if scan.markedPos.Idx >= len(scan.text) {
    return ""
  }

  savedPos := scan.TextPosition
	scan.Pop()
  endIdx := scan.TextPosition.Idx
	scan.TextPosition = savedPos  

	slice := scan.text[scan.markedPos.Idx : endIdx]

	if scan.isComplexSinceMark {
		slice = strings.ReplaceAll(slice, "\r\n", "\n")
		slice = strings.ReplaceAll(slice, "\r", "\n")
		slice = strings.ReplaceAll(slice, "\\\n", "")
	}

	return slice
}

// Stream returns a channel of RuneSpans that are lazily created for the given piece of text.
// The same skipping rules as for Scanner.Pop are applied.
func Stream(text string) <-chan RuneSpan {
	ch := make(chan RuneSpan)
	scan := NewScanner(text)

	go func() {
		defer close(ch)
		for {
			span := scan.PopSpan()
			if span.Rune == EOF {
				return
			}
			ch <- span
		}
	}()
	return ch
}

// ForEach applies the given function for each rule in the provided piece of text.
// The same skipping rules as for Scanner.Pop are applied.
func ForEach(text string, fn func(RuneSpan) bool) {
	scan := NewScanner(text)
	for {
		if span := scan.PopSpan(); span.Rune == EOF || !fn(span) {
			return
		}
	}
}
