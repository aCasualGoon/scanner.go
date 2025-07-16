# Scanner.go
A simple go scanner library.

## Installation
```bash
go get github.com/aCasualGoon/scanner.go
```

## API
### Constants

`ScanEOF`
```go
// ScanEOF represents the synthetic termination rune of a string.
const ScanEOF rune = -1
```

### Types

A `TextPosition` represents a position within a piece of text (string).<br>
> `Idx` <br> Idx is the offset in bytes from the beginning of the string.<br><br>
> `Line` <br> Line is the line component of the position. Can also be seen as the number of line breaks since the beginning of the string plus one.<br><br>
> `Col` <br> Col is the column component of the position. Can also be seen as the number of runes since the last line break plus one.
```go
type TextPosition struct {
    Idx   int
    Line  int
    Col   int
}
```
<br>

A `RuneSpan` represents a rune within text, including the matching positional data.
> `Rune` <br> Rune is the rune.<br><br>
> `Pos` <br> Pos is the position the rune is at.<br><br>
> `End` <br> End is the position one after the rune.
```go
type RuneSpan struct {
  Rune rune
  Pos  TextPosition
  End  TextPosition
}
```
<br>

A `Scanner` holds the data needed to scan a piece of text.
```go
type Scanner struct {...}
```

### Functions

`NewScanner` creates a new scanner for the given piece of text initialized to the default state.
> Default state:<br>
> &nbsp;&nbsp;&nbsp;&nbsp; Idx  = 0<br>
> &nbsp;&nbsp;&nbsp;&nbsp; Line = 1<br>
> &nbsp;&nbsp;&nbsp;&nbsp; Col  = 1
```go
func NewScanner(text string) *Scanner
```
<br>

`Pop` returns the rune at the current scanner position and advances the position to the next rune.
> If the current position is past the end of the text, ScanEOF is returned.<br>
> All line breaks (CR, LF and CRLF) are normalized to LF.<br>
> A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
```go
func (scan *Scanner) Pop() rune
```
<br>

`PopSpan` works like Scanner.Pop but returns the corresponding RuneSpan instead of just the plain rune. 
```go
func (scan *Scanner) PopSpan() RuneSpan
```
<br>

`Peek` returns the rune at the current scanner position without advancing.
> If the current position is past the end of the text, ScanEOF is returned.<br>
> All line breaks (CR, LF and CRLF) are normalized to LF.<br>
> A backslash followed by a line break is skipped and the first rune of the next line is returned instead.
```go
func (scan *Scanner) Peek() rune
```
<br>

`PeekSpan` works like Scanner.Peek but returns the corresponding RuneSpan instead of just the plain rune.
```go
func (scan *Scanner) PeekSpan() RuneSpan
```
<br>

`Mark` marks the rune at the current scanner position to be the first rune in the next Scanner.Slice or Scanner.SliceIncl call.
```go
func (scan *Scanner) Mark()
```
<br>

`Slice` returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (exclusive).
```go
func (scan *Scanner) Slice() string
```
<br>

`SliceIncl` returns the string slice from the last rune marked with Scanner.Mark (inclusive) to the current scanner position (inclusive).
```go
func (scan *Scanner) SliceIncl() string
```
<br>

`Stream` returns a channel of RuneSpans that are lazily created for the given piece of text. 
> The same skipping rules as for Scanner.Pop are applied.
```go
func Stream(text string) <-chan RuneSpan
```
<br>

`ForEach` applies the given function for each rule in the provided piece of text.
> The same skipping rules as for Scanner.Pop are applied.
```go
func ForEach(text string, fn func(RuneSpan) bool)
```
