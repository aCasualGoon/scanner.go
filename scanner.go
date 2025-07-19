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

// A Scanner holds the data needed to scanner a piece of text.
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

// Pos returns the TextPosition the scanner is currently at.
func (scanner *Scanner) Pos() TextPosition {
	return scanner.TextPosition
}

// SetPos hard sets the Scanner to be at the given text position.
func (scanner *Scanner) SetPos(pos TextPosition) {
	scanner.TextPosition = pos
}

// IsEOF returns whether the scanner has moved past the end of the input.
// Positions before the beginning of the input (negative indices) also count as EOF.
func (scanner *Scanner) IsEOF() bool {
  return scanner.Idx < 0 || scanner.Idx >= len(scanner.text)
}

// Pop returns the rune at the current scanner position and advances the position to the next rune.
// If the current position is past the end of the text, ScanEOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) Pop() rune {
	if scanner.IsEOF() {
		return EOF
	}

	r, w := utf8.DecodeRuneInString(scanner.text[scanner.Idx:])

	scanner.Idx += w
	scanner.Col++

	switch r {
	case '\n':
		scanner.Line++
		scanner.Col = 1

	case '\r':
		scanner.Line++
		scanner.Col = 1

		scanner.isComplexSinceMark = true

		// check if part of CRLF. if so, skip LF too.
		if !scanner.IsEOF() {
			if nextR, nextW := utf8.DecodeRuneInString(scanner.text[scanner.Idx:]); nextR == '\n' {
				scanner.Idx += nextW
			}
		}

		// normalize CR and CRLF to LF
		return '\n'

	case '\\':
		savedPos := scanner.TextPosition
		// using pop() automatically handles line break normalization and EOF guard
		if scanner.Pop() != '\n' {
			// we need to reset the scanner position if it's just a regular backslash
			scanner.TextPosition = savedPos
			break
		}

		scanner.isComplexSinceMark = true

		// just return whatever the rune after the escaped line break is
		return scanner.Pop()
	}

	return r
}

// PopSpan works like Scanner.Pop but returns the corresponding RuneSpan instead of just the plain rune.
func (scanner *Scanner) PopSpan() RuneSpan {
	startPos := scanner.TextPosition
	r := scanner.Pop()
	return RuneSpan{
		Rune: r,
		Pos:  startPos,
		End:  scanner.TextPosition,
	}
}

// Peek returns the rune at the current scanner position without advancing.
// If the current position is past the end of the text, ScanEOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) Peek() rune {
	savedPos := scanner.TextPosition
	r := scanner.Pop()
	scanner.TextPosition = savedPos
	return r
}

// PeekSpan works like Scanner.Peek but returns the corresponding RuneSpan instead of just the plain rune.
func (scanner *Scanner) PeekSpan() RuneSpan {
	span := scanner.PopSpan()
	scanner.TextPosition = span.Pos
	return span
}

// Next consumes the rune at the current scanner position and returns the next rune.
// If the current position is past the end of the text, ScanEOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) Next() rune {
	scanner.Pop()
	return scanner.Peek()
}

// NextSpan works like Scanner.Next but returns the corresponding RuneSpan instead of just the plain rune.
func (scanner *Scanner) NextSpan() RuneSpan {
	scanner.Pop()
	return scanner.PeekSpan()
}

// Mark marks the rune at the current scanner position to be the first rune in the next Scanner.Slice or Scanner.SliceIncl call.
func (scanner *Scanner) Mark() {
	scanner.markedPos = scanner.TextPosition
	scanner.isComplexSinceMark = false
}

// Marked returns the TextPosition that was last marked using Scanner.Mark
func (scanner *Scanner) Marked() TextPosition {
  return scanner.markedPos
}

// Slice returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (exclusive).
func (scanner *Scanner) Slice() string {
  if scanner.markedPos.Idx >= len(scanner.text) {
    return ""
  }

	slice := scanner.text[scanner.markedPos.Idx:scanner.Idx]

	if scanner.isComplexSinceMark {
		slice = strings.ReplaceAll(slice, "\r\n", "\n")
		slice = strings.ReplaceAll(slice, "\r", "\n")
		slice = strings.ReplaceAll(slice, "\\\n", "")
	}

	return slice
}

// SliceIncl returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (inclusive).
func (scanner *Scanner) SliceIncl() string {
  if scanner.markedPos.Idx >= len(scanner.text) {
    return ""
  }

  savedPos := scanner.TextPosition
	scanner.Pop()
  endIdx := scanner.TextPosition.Idx
	scanner.TextPosition = savedPos  

	slice := scanner.text[scanner.markedPos.Idx : endIdx]

	if scanner.isComplexSinceMark {
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
	scanner := NewScanner(text)

	go func() {
		defer close(ch)
		for {
			span := scanner.PopSpan()
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
	scanner := NewScanner(text)
	for {
		if span := scanner.PopSpan(); span.Rune == EOF || !fn(span) {
			return
		}
	}
}
