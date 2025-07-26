// Package scanner provides utilities for scanning text (strings).
package scanner

import (
	"strings"
	"unicode/utf8"
)

// EOF represents the synthetic termination rune of a string.
const EOF rune = -1

// TextPosition represents a position within a piece of text (string).
type TextPosition struct {
	// Offset is the offset in bytes from the beginning of the string.
	Offset int
	// Line is the line component of the position. Can also be seen as the number of line breaks since the beginning of the string plus one.
	Line int
	// Column is the column component of the position. Can also be seen as the number of runes since the last line break plus one.
	Col int
}

// A RuneSpan represents a rune within text, including the matching positional data.
type RuneSpan struct {
	// Rune is the rune.
	Rune rune
	// Pos is the position the rune is at.
	Pos TextPosition
	// End is the position after the rune.
	End TextPosition
}

// Scanner scans Unicode text and tracks line/column information.
type Scanner struct {
	TextPosition
	text string

	markedPos          TextPosition
	isComplexSinceMark bool // true if can't be directly sliced
}

// NewScanner creates a new scanner for the given piece of text initialized to the TextPosition at index 0.
func NewScanner(text string) *Scanner {
	return NewScannerAt(text, TextPosition{Offset: 0, Line: 1, Col: 1})
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

// Text returns the text set in the Scanner.
func (scanner *Scanner) Text() string {
	return scanner.text
}

// Pos returns the TextPosition the scanner is currently at.
func (scanner *Scanner) Pos() TextPosition {
	return scanner.TextPosition
}

// SetPos hard sets the Scanner to be at the given TextPosition.
func (scanner *Scanner) SetPos(pos TextPosition) {
	scanner.TextPosition = pos
}

// IsEOF returns whether the scanner has moved past the end of the input.
// Positions before the beginning of the input (negative offset) also count as EOF.
func (scanner *Scanner) IsEOF() bool {
	return scanner.Offset < 0 || scanner.Offset >= len(scanner.text)
}

// Pop returns the rune at the current scanner position and advances the position to the next rune.
// If the current position is past the end of the text, EOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) Pop() rune {
	if scanner.IsEOF() {
		return EOF
	}

	r, w := utf8.DecodeRuneInString(scanner.text[scanner.Offset:])

	scanner.Offset += w
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
			if nextR, nextW := utf8.DecodeRuneInString(scanner.text[scanner.Offset:]); nextR == '\n' {
				scanner.Offset += nextW
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

// PopSpan returns the RuneSpan at the current scanner position and advances the position to the next rune.
// If the current position is past the end of the text, EOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) PopSpan() RuneSpan {
	startPos := scanner.TextPosition
	r := scanner.Pop()
	return RuneSpan{
		Rune: r,
		Pos:  startPos,
		End:  scanner.TextPosition,
	}
}

// PopN returns a string of up to n runes from the current position and advances to the rune after.
// When trying to retrieve runes past the end of the input, the returned string is cut short.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) PopN(n int) string {
	previousMarkedPos := scanner.markedPos

	for range n {
		scanner.Pop()
	}
	text := scanner.Slice()

	scanner.markedPos = previousMarkedPos
	return text
}

// Peek returns the rune at the current scanner position without advancing.
// If the current position is past the end of the text, EOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) Peek() rune {
	savedPos := scanner.TextPosition
	r := scanner.Pop()
	scanner.TextPosition = savedPos
	return r
}

// PeekSpan returns the RuneSpan at the current scanner position without advancing.
// If the current position is past the end of the text, EOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) PeekSpan() RuneSpan {
	span := scanner.PopSpan()
	scanner.TextPosition = span.Pos
	return span
}

// PeekN returns a string of up to n runes from the current position without advancing.
// When trying to retrieve runes past the end of the input, the returned string is cut short.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) PeekN(n int) string {
	previousPos       := scanner.Pos()
	previousMarkedPos := scanner.markedPos

	for range n {
		scanner.Pop()
	}
	text := scanner.Slice()

	scanner.SetPos(previousPos)
	scanner.markedPos = previousMarkedPos
	return text
}

// Next consumes the rune at the current scanner position and returns the next rune.
// If the current position is past the end of the text, EOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
func (scanner *Scanner) Next() rune {
	scanner.Pop()
	return scanner.Peek()
}

// NextSpan consumes the RuneSpan at the current scanner position and returns the next rune.
// If the current position is past the end of the text, EOF is returned.
// All line breaks (CR, LF and CRLF) are normalized to LF.
// A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
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
	if scanner.markedPos.Offset >= len(scanner.text) {
		return ""
	}

	slice := scanner.text[scanner.markedPos.Offset:scanner.Offset]

	if scanner.isComplexSinceMark {
		slice = strings.ReplaceAll(slice, "\r\n", "\n")
		slice = strings.ReplaceAll(slice, "\r", "\n")
		slice = strings.ReplaceAll(slice, "\\\n", "")
	}

	return slice
}

// SliceInc returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (inclusive).
func (scanner *Scanner) SliceInc() string {
	if scanner.markedPos.Offset >= len(scanner.text) {
		return ""
	}

	savedPos := scanner.TextPosition
	scanner.Pop()
	endIdx := scanner.TextPosition.Offset
	scanner.TextPosition = savedPos

	slice := scanner.text[scanner.markedPos.Offset:endIdx]

	if scanner.isComplexSinceMark {
		slice = strings.ReplaceAll(slice, "\r\n", "\n")
		slice = strings.ReplaceAll(slice, "\r", "\n")
		slice = strings.ReplaceAll(slice, "\\\n", "")
	}

	return slice
}

// Stream returns a channel of RuneSpans that are lazily created for the given piece of text.
// The same skipping rules as for Scanner.Pop are applied.
// The use of a channel may add allocation overhead, prefer manual iteration for performance critical applications.
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
