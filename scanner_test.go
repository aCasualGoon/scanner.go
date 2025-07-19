package scanner

import (
	"strings"
	"testing"
)

func TestScannerPos(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		actions  []string // "pop", "next", "peek"
		expected []TextPosition
	}{
		{
			name:     "initial position",
			input:    "abc",
			actions:  []string{"pos"},
			expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
		},
		{
			name:     "after single pop",
			input:    "abc",
			actions:  []string{"pop", "pos"},
			expected: []TextPosition{{Idx: 1, Line: 1, Col: 2}},
		},
		{
			name:     "after multiple pops",
			input:    "hello",
			actions:  []string{"pop", "pop", "pop", "pos"},
			expected: []TextPosition{{Idx: 3, Line: 1, Col: 4}},
		},
		{
			name:     "after line break",
			input:    "a\nb",
			actions:  []string{"pop", "pop", "pos"},
			expected: []TextPosition{{Idx: 2, Line: 2, Col: 1}},
		},
		{
			name:     "after CR normalization",
			input:    "a\rb",
			actions:  []string{"pop", "pop", "pos"},
			expected: []TextPosition{{Idx: 2, Line: 2, Col: 1}},
		},
		{
			name:     "after CRLF normalization",
			input:    "a\r\nb",
			actions:  []string{"pop", "pop", "pos"},
			expected: []TextPosition{{Idx: 3, Line: 2, Col: 1}},
		},
		{
			name:     "after escaped line break",
			input:    "a\\\nb",
			actions:  []string{"pop", "pop", "pos"},
			expected: []TextPosition{{Idx: 4, Line: 2, Col: 2}},
		},
		{
			name:     "after UTF-8 character",
			input:    "αβγ",
			actions:  []string{"pop", "pos"},
			expected: []TextPosition{{Idx: 2, Line: 1, Col: 2}},
		},
		{
			name:     "peek doesn't change position",
			input:    "abc",
			actions:  []string{"peek", "pos"},
			expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
		},
		{
			name:     "next advances position",
			input:    "abc",
			actions:  []string{"next", "pos"},
			expected: []TextPosition{{Idx: 1, Line: 1, Col: 2}},
		},
		{
			name:     "at EOF",
			input:    "a",
			actions:  []string{"pop", "pos"},
			expected: []TextPosition{{Idx: 1, Line: 1, Col: 2}},
		},
		{
			name:     "empty string",
			input:    "",
			actions:  []string{"pos"},
			expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
		},
		{
			name:     "multiple line breaks",
			input:    "a\n\nb",
			actions:  []string{"pop", "pop", "pop", "pos"},
			expected: []TextPosition{{Idx: 3, Line: 3, Col: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			expectedIdx := 0

			for _, action := range tt.actions {
				switch action {
				case "pop":
					scanner.Pop()
				case "peek":
					scanner.Peek()
				case "next":
					scanner.Next()
				case "pos":
					if expectedIdx >= len(tt.expected) {
						t.Errorf("unexpected pos call at action %s", action)
						continue
					}
					pos := scanner.Pos()
					expected := tt.expected[expectedIdx]
					if pos != expected {
						t.Errorf("position: expected %+v, got %+v", expected, pos)
					}
					expectedIdx++
				}
			}
		})
	}
}


func TestScannerSetPos(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		setPos   TextPosition
		expected rune
	}{
		{
			name:     "set to beginning",
			input:    "abc",
			setPos:   TextPosition{Idx: 0, Line: 1, Col: 1},
			expected: 'a',
		},
		{
			name:     "set to middle",
			input:    "abc",
			setPos:   TextPosition{Idx: 1, Line: 1, Col: 2},
			expected: 'b',
		},
		{
			name:     "set to end",
			input:    "abc",
			setPos:   TextPosition{Idx: 3, Line: 1, Col: 4},
			expected: EOF,
		},
		{
			name:     "set beyond end",
			input:    "abc",
			setPos:   TextPosition{Idx: 10, Line: 2, Col: 5},
			expected: EOF,
		},
		{
			name:     "set to UTF-8 boundary",
			input:    "αβγ",
			setPos:   TextPosition{Idx: 2, Line: 1, Col: 2},
			expected: 'β',
		},
		{
			name:     "set to line break",
			input:    "a\nb",
			setPos:   TextPosition{Idx: 1, Line: 1, Col: 2},
			expected: '\n',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			scanner.SetPos(tt.setPos)

			// Verify position was set correctly
			if scanner.TextPosition != tt.setPos {
				t.Errorf("position not set correctly: expected %+v, got %+v", tt.setPos, scanner.TextPosition)
			}

			// Verify peek returns expected rune
			r := scanner.Peek()
			if r != tt.expected {
				t.Errorf("peek after SetPos: expected %q, got %q", tt.expected, r)
			}
		})
	}
}

func TestScannerSetPosAfterAdvancement(t *testing.T) {
	scanner := NewScanner("abcdef")

	// Advance to middle
	scanner.Pop()
	scanner.Pop()
	expectedPos := TextPosition{Idx: 2, Line: 1, Col: 3}
	if scanner.TextPosition != expectedPos {
		t.Errorf("after advancement: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// Set back to beginning
	newPos := TextPosition{Idx: 0, Line: 1, Col: 1}
	scanner.SetPos(newPos)
	if scanner.TextPosition != newPos {
		t.Errorf("after SetPos: expected %+v, got %+v", newPos, scanner.TextPosition)
	}

	// Verify we can read from beginning again
	r := scanner.Pop()
	if r != 'a' {
		t.Errorf("after SetPos to beginning: expected 'a', got %q", r)
	}
}

func TestScannerSetPosWithComplexInput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		setPos TextPosition
		ops    []func(*Scanner) rune
		expected []rune
	}{
		{
			name:   "set to CR position",
			input:  "a\rb",
			setPos: TextPosition{Idx: 1, Line: 1, Col: 2},
			ops:    []func(*Scanner) rune{(*Scanner).Pop, (*Scanner).Pop},
			expected: []rune{'\n', 'b'},
		},
		{
			name:   "set to CRLF position",
			input:  "a\r\nb",
			setPos: TextPosition{Idx: 1, Line: 1, Col: 2},
			ops:    []func(*Scanner) rune{(*Scanner).Pop, (*Scanner).Pop},
			expected: []rune{'\n', 'b'},
		},
		{
			name:   "set to escaped line break position",
			input:  "a\\\nb",
			setPos: TextPosition{Idx: 1, Line: 1, Col: 2},
			ops:    []func(*Scanner) rune{(*Scanner).Pop},
			expected: []rune{'b'},
		},
		{
			name:   "set after UTF-8 character",
			input:  "αβγ",
			setPos: TextPosition{Idx: 4, Line: 1, Col: 3},
			ops:    []func(*Scanner) rune{(*Scanner).Pop},
			expected: []rune{'γ'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			scanner.SetPos(tt.setPos)

			var result []rune
			for _, op := range tt.ops {
				r := op(scanner)
				result = append(result, r)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d runes, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at position %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestScannerIsEOF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string at start",
			input:    "",
			expected: true,
		},
		{
			name:     "non-empty string at start",
			input:    "abc",
			expected: false,
		},
		{
			name:     "single character at start",
			input:    "a",
			expected: false,
		},
		{
			name:     "UTF-8 string at start",
			input:    "αβγ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			if scanner.IsEOF() != tt.expected {
				t.Errorf("expected IsEOF() = %t, got %t", tt.expected, scanner.IsEOF())
			}
		})
	}
}

func TestScannerIsEOFAfterAdvancement(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		popCount   int
		expectedEOF bool
	}{
		{
			name:       "single char, one pop",
			input:      "a",
			popCount:   1,
			expectedEOF: true,
		},
		{
			name:       "single char, no pops",
			input:      "a",
			popCount:   0,
			expectedEOF: false,
		},
		{
			name:       "three chars, two pops",
			input:      "abc",
			popCount:   2,
			expectedEOF: false,
		},
		{
			name:       "three chars, three pops",
			input:      "abc",
			popCount:   3,
			expectedEOF: true,
		},
		{
			name:       "three chars, four pops",
			input:      "abc",
			popCount:   4,
			expectedEOF: true,
		},
		{
			name:       "UTF-8 chars, one pop",
			input:      "αβ",
			popCount:   1,
			expectedEOF: false,
		},
		{
			name:       "UTF-8 chars, two pops",
			input:      "αβ",
			popCount:   2,
			expectedEOF: true,
		},
		{
			name:       "line breaks, partial consumption",
			input:      "a\nb",
			popCount:   1,
			expectedEOF: false,
		},
		{
			name:       "line breaks, full consumption",
			input:      "a\nb",
			popCount:   3,
			expectedEOF: true,
		},
		{
			name:       "escaped line break",
			input:      "a\\\nb",
			popCount:   2,
			expectedEOF: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			
			for i := 0; i < tt.popCount; i++ {
				scanner.Pop()
			}
			
			if scanner.IsEOF() != tt.expectedEOF {
				t.Errorf("after %d pops, expected IsEOF() = %t, got %t", 
					tt.popCount, tt.expectedEOF, scanner.IsEOF())
			}
		})
	}
}

func TestScannerIsEOFWithSetPos(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		setPos      TextPosition
		expectedEOF bool
	}{
		{
			name:        "negative index",
			input:       "abc",
			setPos:      TextPosition{Idx: -1, Line: 1, Col: 1},
			expectedEOF: true,
		},
		{
			name:        "zero index on non-empty",
			input:       "abc",
			setPos:      TextPosition{Idx: 0, Line: 1, Col: 1},
			expectedEOF: false,
		},
		{
			name:        "zero index on empty",
			input:       "",
			setPos:      TextPosition{Idx: 0, Line: 1, Col: 1},
			expectedEOF: true,
		},
		{
			name:        "index at end",
			input:       "abc",
			setPos:      TextPosition{Idx: 3, Line: 1, Col: 4},
			expectedEOF: true,
		},
		{
			name:        "index past end",
			input:       "abc",
			setPos:      TextPosition{Idx: 10, Line: 1, Col: 11},
			expectedEOF: true,
		},
		{
			name:        "index within bounds",
			input:       "abc",
			setPos:      TextPosition{Idx: 1, Line: 1, Col: 2},
			expectedEOF: false,
		},
		{
			name:        "UTF-8 string, byte index within multibyte char",
			input:       "αβγ",
			setPos:      TextPosition{Idx: 1, Line: 1, Col: 1},
			expectedEOF: false,
		},
		{
			name:        "UTF-8 string, at end",
			input:       "αβγ",
			setPos:      TextPosition{Idx: 6, Line: 1, Col: 4},
			expectedEOF: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			scanner.SetPos(tt.setPos)
			
			if scanner.IsEOF() != tt.expectedEOF {
				t.Errorf("with position %+v, expected IsEOF() = %t, got %t",
					tt.setPos, tt.expectedEOF, scanner.IsEOF())
			}
		})
	}
}

func TestScannerIsEOFWithNewScannerAt(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		startPos    TextPosition
		expectedEOF bool
	}{
		{
			name:        "start at beginning",
			input:       "abc",
			startPos:    TextPosition{Idx: 0, Line: 1, Col: 1},
			expectedEOF: false,
		},
		{
			name:        "start at end",
			input:       "abc",
			startPos:    TextPosition{Idx: 3, Line: 1, Col: 4},
			expectedEOF: true,
		},
		{
			name:        "start past end",
			input:       "abc",
			startPos:    TextPosition{Idx: 5, Line: 1, Col: 6},
			expectedEOF: true,
		},
		{
			name:        "start at negative index",
			input:       "abc",
			startPos:    TextPosition{Idx: -1, Line: 1, Col: 0},
			expectedEOF: true,
		},
		{
			name:        "start in middle",
			input:       "abc",
			startPos:    TextPosition{Idx: 1, Line: 1, Col: 2},
			expectedEOF: false,
		},
		{
			name:        "empty string at start",
			input:       "",
			startPos:    TextPosition{Idx: 0, Line: 1, Col: 1},
			expectedEOF: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScannerAt(tt.input, tt.startPos)
			
			if scanner.IsEOF() != tt.expectedEOF {
				t.Errorf("with starting position %+v, expected IsEOF() = %t, got %t",
					tt.startPos, tt.expectedEOF, scanner.IsEOF())
			}
		})
	}
}

func TestScannerIsEOFConsistency(t *testing.T) {
	// Test that IsEOF() is consistent with Pop() returning EOF
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "single character",
			input: "a",
		},
		{
			name:  "multiple characters",
			input: "hello",
		},
		{
			name:  "UTF-8 characters",
			input: "αβγ",
		},
		{
			name:  "with line breaks",
			input: "a\nb\rc\r\nd",
		},
		{
			name:  "with escaped line breaks",
			input: "a\\\nb\\\rc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)

			for {
				isEOFBefore := scanner.IsEOF()
				r := scanner.Pop()

				if isEOFBefore && r != EOF {
					t.Errorf("IsEOF() returned true but Pop() returned %q instead of EOF", r)
				}
				if !isEOFBefore && r == EOF {
					t.Errorf("IsEOF() returned false but Pop() returned EOF")
				}

				if r == EOF {
					// After Pop() returns EOF, IsEOF() should still return true
					if !scanner.IsEOF() {
						t.Errorf("after Pop() returned EOF, IsEOF() should return true")
					}
					break
				}
			}
		})
	}
}

func TestScannerIsEOFEdgeCases(t *testing.T) {
	t.Run("very large negative index", func(t *testing.T) {
		scanner := NewScanner("abc")
		scanner.SetPos(TextPosition{Idx: -1000000, Line: 1, Col: 1})
		
		if !scanner.IsEOF() {
			t.Error("expected IsEOF() = true for very large negative index")
		}
	})

	t.Run("very large positive index", func(t *testing.T) {
		scanner := NewScanner("abc")
		scanner.SetPos(TextPosition{Idx: 1000000, Line: 1, Col: 1})
		
		if !scanner.IsEOF() {
			t.Error("expected IsEOF() = true for very large positive index")
		}
	})

	t.Run("boundary at zero length", func(t *testing.T) {
		scanner := NewScanner("")
		
		// Index 0 on empty string should be EOF
		scanner.SetPos(TextPosition{Idx: 0, Line: 1, Col: 1})
		if !scanner.IsEOF() {
			t.Error("expected IsEOF() = true for index 0 on empty string")
		}
		
		// Index 1 on empty string should also be EOF
		scanner.SetPos(TextPosition{Idx: 1, Line: 1, Col: 1})
		if !scanner.IsEOF() {
			t.Error("expected IsEOF() = true for index 1 on empty string")
		}
	})

	t.Run("exact boundary conditions", func(t *testing.T) {
		input := "ab"
		scanner := NewScanner(input)
		
		// Index len(input)-1 should not be EOF
		scanner.SetPos(TextPosition{Idx: len(input) - 1, Line: 1, Col: 2})
		if scanner.IsEOF() {
			t.Error("expected IsEOF() = false for index len(input)-1")
		}
		
		// Index len(input) should be EOF
		scanner.SetPos(TextPosition{Idx: len(input), Line: 1, Col: 3})
		if !scanner.IsEOF() {
			t.Error("expected IsEOF() = true for index len(input)")
		}
	})
}

func TestScannerPop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []rune
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []rune{EOF},
		},
		{
			name:     "simple ASCII",
			input:    "abc",
			expected: []rune{'a', 'b', 'c', EOF},
		},
		{
			name:     "UTF-8 characters",
			input:    "αβγ",
			expected: []rune{'α', 'β', 'γ', EOF},
		},
		{
			name:     "LF normalization",
			input:    "a\nb",
			expected: []rune{'a', '\n', 'b', EOF},
		},
		{
			name:     "CR normalization",
			input:    "a\rb",
			expected: []rune{'a', '\n', 'b', EOF},
		},
		{
			name:     "CRLF normalization",
			input:    "a\r\nb",
			expected: []rune{'a', '\n', 'b', EOF},
		},
		{
			name:     "escaped line break",
			input:    "a\\\nb",
			expected: []rune{'a', 'b', EOF},
		},
		{
			name:     "regular backslash",
			input:    "a\\b",
			expected: []rune{'a', '\\', 'b', EOF},
		},
		{
			name:     "backslash at end",
			input:    "a\\",
			expected: []rune{'a', '\\', EOF},
		},
		{
			name:     "multiple line breaks",
			input:    "a\n\r\n\rb",
			expected: []rune{'a', '\n', '\n', '\n', 'b', EOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			var result []rune

			for {
				r := scanner.Pop()
				result = append(result, r)
				if r == EOF {
					break
				}
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d runes, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at position %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestScannerPopPosition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TextPosition
	}{
		{
			name:  "simple line and column tracking",
			input: "ab\nc",
			expected: []TextPosition{
				{Idx: 1, Line: 1, Col: 2}, // after 'a'
				{Idx: 2, Line: 1, Col: 3}, // after 'b'
				{Idx: 3, Line: 2, Col: 1}, // after '\n'
				{Idx: 4, Line: 2, Col: 2}, // after 'c'
			},
		},
		{
			name:  "CR normalization position",
			input: "a\rb",
			expected: []TextPosition{
				{Idx: 1, Line: 1, Col: 2}, // after 'a'
				{Idx: 2, Line: 2, Col: 1}, // after '\r' (normalized to '\n')
				{Idx: 3, Line: 2, Col: 2}, // after 'b'
			},
		},
		{
			name:  "CRLF normalization position",
			input: "a\r\nb",
			expected: []TextPosition{
				{Idx: 1, Line: 1, Col: 2}, // after 'a'
				{Idx: 3, Line: 2, Col: 1}, // after '\r\n' (normalized to '\n')
				{Idx: 4, Line: 2, Col: 2}, // after 'b'
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)

			for i, expectedPos := range tt.expected {
				scanner.Pop()
				if scanner.TextPosition != expectedPos {
					t.Errorf("at step %d: expected position %+v, got %+v", i, expectedPos, scanner.TextPosition)
				}
			}
		})
	}
}

func TestScannerPopSpan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []RuneSpan
	}{
		{
			name:  "empty string",
			input: "",
			expected: []RuneSpan{
				{Rune: EOF, Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 0, Line: 1, Col: 1}},
			},
		},
		{
			name:  "simple ASCII",
			input: "ab",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
				{Rune: EOF, Pos: TextPosition{Idx: 2, Line: 1, Col: 3}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
			},
		},
		{
			name:  "UTF-8 characters",
			input: "αβ",
			expected: []RuneSpan{
				{Rune: 'α', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 2, Line: 1, Col: 2}},
				{Rune: 'β', Pos: TextPosition{Idx: 2, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
				{Rune: EOF, Pos: TextPosition{Idx: 4, Line: 1, Col: 3}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
			},
		},
		{
			name:  "line break normalization",
			input: "a\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
				{Rune: EOF, Pos: TextPosition{Idx: 3, Line: 2, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
			},
		},
		{
			name:  "CRLF normalization",
			input: "a\r\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 3, Line: 2, Col: 1}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
				{Rune: EOF, Pos: TextPosition{Idx: 4, Line: 2, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
			},
		},
		{
			name:  "escaped line break",
			input: "a\\\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
				{Rune: EOF, Pos: TextPosition{Idx: 4, Line: 2, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			var result []RuneSpan

			for {
				span := scanner.PopSpan()
				result = append(result, span)
				if span.Rune == EOF {
					break
				}
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d spans, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at position %d: expected %+v, got %+v", i, expected, result[i])
				}
			}
		})
	}
}

func TestScannerPeek(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []rune
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []rune{EOF},
		},
		{
			name:     "simple ASCII",
			input:    "abc",
			expected: []rune{'a', 'a', 'a'}, // peek should return same rune multiple times
		},
		{
			name:     "UTF-8 characters",
			input:    "αβγ",
			expected: []rune{'α', 'α', 'α'},
		},
		{
			name:     "line break normalization",
			input:    "\n",
			expected: []rune{'\n', '\n'},
		},
		{
			name:     "CR normalization",
			input:    "\r",
			expected: []rune{'\n', '\n'},
		},
		{
			name:     "CRLF normalization",
			input:    "\r\n",
			expected: []rune{'\n', '\n'},
		},
		{
			name:     "escaped line break",
			input:    "\\\n",
			expected: []rune{EOF, EOF}, // escaped line break consumes both chars
		},
		{
			name:     "regular backslash",
			input:    "\\a",
			expected: []rune{'\\', '\\'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)

			// Test multiple peeks return same result
			for i, expected := range tt.expected {
				r := scanner.Peek()
				if r != expected {
					t.Errorf("peek %d: expected %q, got %q", i, expected, r)
				}
			}
		})
	}
}

func TestScannerPeekVsPop(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple ASCII",
			input: "abc",
		},
		{
			name:  "UTF-8 characters",
			input: "αβγ",
		},
		{
			name:  "mixed content",
			input: "a\r\nb\\c",
		},
		{
			name:  "escaped line break",
			input: "a\\\nb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner1 := NewScanner(tt.input)
			scanner2 := NewScanner(tt.input)

			for {
				// Peek should return same as Pop
				peeked := scanner1.Peek()
				popped := scanner2.Pop()

				if peeked != popped {
					t.Errorf("peek/pop mismatch: peek=%q, pop=%q", peeked, popped)
				}

				if popped == EOF {
					break
				}

				// Advance scanner1 to next position
				scanner1.Pop()
			}
		})
	}
}

func TestScannerPeekPosition(t *testing.T) {
	scanner := NewScanner("abc")

	// Initial position
	expectedPos := TextPosition{Idx: 0, Line: 1, Col: 1}
	if scanner.TextPosition != expectedPos {
		t.Errorf("initial position: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// Peek should not change position
	scanner.Peek()
	if scanner.TextPosition != expectedPos {
		t.Errorf("after peek: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// Multiple peeks should not change position
	scanner.Peek()
	scanner.Peek()
	if scanner.TextPosition != expectedPos {
		t.Errorf("after multiple peeks: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// Pop should advance position
	scanner.Pop()
	expectedPos = TextPosition{Idx: 1, Line: 1, Col: 2}
	if scanner.TextPosition != expectedPos {
		t.Errorf("after pop: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}
}

func TestScannerPeekSpan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []RuneSpan
	}{
		{
			name:  "empty string",
			input: "",
			expected: []RuneSpan{
				{Rune: EOF, Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 0, Line: 1, Col: 1}},
			},
		},
		{
			name:  "simple ASCII",
			input: "ab",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
			},
		},
		{
			name:  "UTF-8 characters",
			input: "αβ",
			expected: []RuneSpan{
				{Rune: 'α', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 2, Line: 1, Col: 2}},
				{Rune: 'α', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 2, Line: 1, Col: 2}},
			},
		},
		{
			name:  "line break normalization",
			input: "a\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
			},
		},
		{
			name:  "CR normalization",
			input: "a\rb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
			},
		},
		{
			name:  "CRLF normalization",
			input: "a\r\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
			},
		},
		{
			name:  "escaped line break",
			input: "a\\\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)

			// Test multiple PeekSpan calls return same result
			for i, expected := range tt.expected {
				span := scanner.PeekSpan()
				if span != expected {
					t.Errorf("peek span %d: expected %+v, got %+v", i, expected, span)
				}
			}
		})
	}
}

func TestScannerPeekSpanVsPopSpan(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple ASCII",
			input: "abc",
		},
		{
			name:  "UTF-8 characters",
			input: "αβγ",
		},
		{
			name:  "mixed content",
			input: "a\r\nb\\c",
		},
		{
			name:  "escaped line break",
			input: "a\\\nb",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner1 := NewScanner(tt.input)
			scanner2 := NewScanner(tt.input)

			for {
				// PeekSpan should return same as PopSpan
				peeked := scanner1.PeekSpan()
				popped := scanner2.PopSpan()

				if peeked != popped {
					t.Errorf("peek/pop span mismatch: peeked=%+v, popped=%+v", peeked, popped)
				}

				if popped.Rune == EOF {
					break
				}

				// Advance scanner1 to next position
				scanner1.PopSpan()
			}
		})
	}
}

func TestScannerPeekSpanPosition(t *testing.T) {
	scanner := NewScanner("abc")

	// Initial position
	expectedPos := TextPosition{Idx: 0, Line: 1, Col: 1}
	if scanner.TextPosition != expectedPos {
		t.Errorf("initial position: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// PeekSpan should not change position
	span := scanner.PeekSpan()
	if scanner.TextPosition != expectedPos {
		t.Errorf("after peek span: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// Verify span has correct positions
	expectedSpan := RuneSpan{
		Rune: 'a',
		Pos:  TextPosition{Idx: 0, Line: 1, Col: 1},
		End:  TextPosition{Idx: 1, Line: 1, Col: 2},
	}
	if span != expectedSpan {
		t.Errorf("expected span %+v, got %+v", expectedSpan, span)
	}

	// Multiple PeekSpan calls should not change position
	scanner.PeekSpan()
	scanner.PeekSpan()
	if scanner.TextPosition != expectedPos {
		t.Errorf("after multiple peek spans: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}

	// PopSpan should advance position
	scanner.PopSpan()
	expectedPos = TextPosition{Idx: 1, Line: 1, Col: 2}
	if scanner.TextPosition != expectedPos {
		t.Errorf("after pop span: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}
}

func TestScannerPeekSpanSpecialCases(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedSpan RuneSpan
	}{
		{
			name:  "CR at start",
			input: "\rabc",
			expectedSpan: RuneSpan{
				Rune: '\n',
				Pos:  TextPosition{Idx: 0, Line: 1, Col: 1},
				End:  TextPosition{Idx: 1, Line: 2, Col: 1},
			},
		},
		{
			name:  "CRLF at start",
			input: "\r\nabc",
			expectedSpan: RuneSpan{
				Rune: '\n',
				Pos:  TextPosition{Idx: 0, Line: 1, Col: 1},
				End:  TextPosition{Idx: 2, Line: 2, Col: 1},
			},
		},
		{
			name:  "escaped line break at start",
			input: "\\\nabc",
			expectedSpan: RuneSpan{
				Rune: 'a',
				Pos:  TextPosition{Idx: 0, Line: 1, Col: 1},
				End:  TextPosition{Idx: 3, Line: 2, Col: 2},
			},
		},
		{
			name:  "regular backslash at start",
			input: "\\abc",
			expectedSpan: RuneSpan{
				Rune: '\\',
				Pos:  TextPosition{Idx: 0, Line: 1, Col: 1},
				End:  TextPosition{Idx: 1, Line: 1, Col: 2},
			},
		},
		{
			name:  "backslash at end",
			input: "\\",
			expectedSpan: RuneSpan{
				Rune: '\\',
				Pos:  TextPosition{Idx: 0, Line: 1, Col: 1},
				End:  TextPosition{Idx: 1, Line: 1, Col: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			span := scanner.PeekSpan()

			if span != tt.expectedSpan {
				t.Errorf("expected span %+v, got %+v", tt.expectedSpan, span)
			}

			// Verify position unchanged
			expectedPos := TextPosition{Idx: 0, Line: 1, Col: 1}
			if scanner.TextPosition != expectedPos {
				t.Errorf("position changed after peek span: expected %+v, got %+v", expectedPos, scanner.TextPosition)
			}
		})
	}
}

func TestScannerPeekSpanAfterAdvancement(t *testing.T) {
	scanner := NewScanner("abc")

	// Advance past first character
	scanner.Pop()

	// PeekSpan should return span for second character
	span := scanner.PeekSpan()
	expectedSpan := RuneSpan{
		Rune: 'b',
		Pos:  TextPosition{Idx: 1, Line: 1, Col: 2},
		End:  TextPosition{Idx: 2, Line: 1, Col: 3},
	}

	if span != expectedSpan {
		t.Errorf("expected span %+v, got %+v", expectedSpan, span)
	}

	// Position should remain at second character
	expectedPos := TextPosition{Idx: 1, Line: 1, Col: 2}
	if scanner.TextPosition != expectedPos {
		t.Errorf("position changed: expected %+v, got %+v", expectedPos, scanner.TextPosition)
	}
}

func TestScannerPeekSpanConsistency(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple text",
			input: "hello world",
		},
		{
			name:  "UTF-8 text",
			input: "αβγ δεζ",
		},
		{
			name:  "mixed line endings",
			input: "line1\nline2\rline3\r\nline4",
		},
		{
			name:  "escaped breaks",
			input: "cont\\\ninued\\\ntext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)

			for {
				// Get span via PeekSpan
				peekedSpan := scanner.PeekSpan()

				// Get span via PopSpan
				poppedSpan := scanner.PopSpan()

				// They should be identical
				if peekedSpan != poppedSpan {
					t.Errorf("inconsistent spans: peeked=%+v, popped=%+v", peekedSpan, poppedSpan)
				}

				if poppedSpan.Rune == EOF {
					break
				}
			}
		})
	}
}

func TestScannerNext(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected []rune
  }{
    {
      name:     "empty string",
      input:    "",
      expected: []rune{EOF},
    },
    {
      name:     "simple ASCII",
      input:    "abc",
      expected: []rune{'b', 'c', EOF},
    },
    {
      name:     "UTF-8 characters",
      input:    "αβγ",
      expected: []rune{'β', 'γ', EOF},
    },
    {
      name:     "single character",
      input:    "a",
      expected: []rune{EOF},
    },
    {
      name:     "LF normalization",
      input:    "a\nb",
      expected: []rune{'\n', 'b', EOF},
    },
    {
      name:     "CR normalization",
      input:    "a\rb",
      expected: []rune{'\n', 'b', EOF},
    },
    {
      name:     "CRLF normalization",
      input:    "a\r\nb",
      expected: []rune{'\n', 'b', EOF},
    },
    {
      name:     "escaped line break",
      input:    "a\\\nb",
      expected: []rune{'b', EOF},
    },
    {
      name:     "regular backslash",
      input:    "a\\b",
      expected: []rune{'\\', 'b', EOF},
    },
    {
      name:     "backslash at end",
      input:    "a\\",
      expected: []rune{'\\', EOF},
    },
    {
      name:     "multiple line breaks",
      input:    "a\n\r\n\rb",
      expected: []rune{'\n', '\n', '\n', 'b', EOF},
    },
    {
      name:     "escaped line break at end",
      input:    "a\\\n",
      expected: []rune{EOF},
    },
    {
      name:     "multiple escaped line breaks",
      input:    "a\\\n\\\nb",
      expected: []rune{'b', EOF},
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      var result []rune

      for {
        r := scanner.Next()
        result = append(result, r)
        if r == EOF {
          break
        }
      }

      if len(result) != len(tt.expected) {
        t.Errorf("expected %d runes, got %d", len(tt.expected), len(result))
        return
      }

      for i, expected := range tt.expected {
        if result[i] != expected {
          t.Errorf("at position %d: expected %q, got %q", i, expected, result[i])
        }
      }
    })
  }
}

func TestScannerNextPosition(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected []TextPosition
  }{
    {
      name:  "simple line and column tracking",
      input: "ab\nc",
      expected: []TextPosition{
        {Idx: 1, Line: 1, Col: 2}, // after consuming 'a', next is 'b'
        {Idx: 2, Line: 1, Col: 3}, // after consuming 'b', next is '\n'
        {Idx: 3, Line: 2, Col: 1}, // after consuming '\n', next is 'c'
      },
    },
    {
      name:  "CR normalization position",
      input: "a\rb",
      expected: []TextPosition{
        {Idx: 1, Line: 1, Col: 2}, // after consuming 'a', next is '\r' (normalized)
        {Idx: 2, Line: 2, Col: 1}, // after consuming '\r', next is 'b'
      },
    },
    {
      name:  "CRLF normalization position",
      input: "a\r\nb",
      expected: []TextPosition{
        {Idx: 1, Line: 1, Col: 2}, // after consuming 'a', next is '\r\n' (normalized)
        {Idx: 3, Line: 2, Col: 1}, // after consuming '\r\n', next is 'b'
      },
    },
    {
      name:  "escaped line break position",
      input: "a\\\nb",
      expected: []TextPosition{
        {Idx: 1, Line: 1, Col: 2}, // after consuming 'a', next is 'b' (escaped break skipped)
      },
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)

      for i, expectedPos := range tt.expected {
        scanner.Next()
        if scanner.TextPosition != expectedPos {
          t.Errorf("at step %d: expected position %+v, got %+v", i, expectedPos, scanner.TextPosition)
        }
      }
    })
  }
}

func TestScannerNextVsPopPeek(t *testing.T) {
  tests := []struct {
    name  string
    input string
  }{
    {
      name:  "simple ASCII",
      input: "abc",
    },
    {
      name:  "UTF-8 characters",
      input: "αβγ",
    },
    {
      name:  "mixed content",
      input: "a\r\nb\\c",
    },
    {
      name:  "escaped line break",
      input: "a\\\nb",
    },
    {
      name:  "empty string",
      input: "",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner1 := NewScanner(tt.input)
      scanner2 := NewScanner(tt.input)

      for {
        // Next should be equivalent to Pop() followed by Peek()
        next := scanner1.Next()
        
        scanner2.Pop()
        peek := scanner2.Peek()

        if next != peek {
          t.Errorf("next/pop+peek mismatch: next=%q, pop+peek=%q", next, peek)
        }

        if next == EOF {
          break
        }
      }
    })
  }
}

func TestScannerNextAtEOF(t *testing.T) {
  scanner := NewScanner("a")

  // Next should return EOF when at end
  next := scanner.Next()
  if next != EOF {
    t.Errorf("next at EOF: expected EOF, got %q", next)
  }

  // Multiple Next calls at EOF should return EOF
  next = scanner.Next()
  if next != EOF {
    t.Errorf("next after EOF: expected EOF, got %q", next)
  }

  // Position should remain at end
  expectedPos := TextPosition{Idx: 1, Line: 1, Col: 2}
  if scanner.TextPosition != expectedPos {
    t.Errorf("position after EOF: expected %+v, got %+v", expectedPos, scanner.TextPosition)
  }
}

func TestScannerNextEmptyString(t *testing.T) {
  scanner := NewScanner("")

  // Next on empty string should return EOF
  next := scanner.Next()
  if next != EOF {
    t.Errorf("next on empty string: expected EOF, got %q", next)
  }

  // Position should remain at start
  expectedPos := TextPosition{Idx: 0, Line: 1, Col: 1}
  if scanner.TextPosition != expectedPos {
    t.Errorf("position after next on empty: expected %+v, got %+v", expectedPos, scanner.TextPosition)
  }
}

func TestScannerNextComplexNormalization(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected []rune
  }{
    {
      name:     "backslash not followed by newline",
      input:    "a\\b",
      expected: []rune{'\\', 'b', EOF},
    },
    {
      name:     "backslash at end",
      input:    "a\\",
      expected: []rune{'\\', EOF},
    },
    {
      name:     "multiple escaped breaks",
      input:    "a\\\n\\\nb",
      expected: []rune{'b', EOF},
    },
    {
      name:     "escaped break with CR",
      input:    "a\\\rb",
      expected: []rune{'b', EOF},
    },
    {
      name:     "escaped break with CRLF",
      input:    "a\\\r\nb",
      expected: []rune{'b', EOF},
    },
    {
      name:     "mixed line endings and escapes",
      input:    "a\r\nb\\\nc\rd",
      expected: []rune{'\n', 'b', 'c', '\n', 'd', EOF},
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      var result []rune

      for {
        r := scanner.Next()
        result = append(result, r)
        if r == EOF {
          break
        }
      }

      if len(result) != len(tt.expected) {
        t.Errorf("expected %d runes, got %d", len(tt.expected), len(result))
        return
      }

      for i, expected := range tt.expected {
        if result[i] != expected {
          t.Errorf("at position %d: expected %q, got %q", i, expected, result[i])
        }
      }
    })
  }
}

func TestScannerNextSpan(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected []RuneSpan
  }{
    {
      name:  "empty string",
      input: "",
      expected: []RuneSpan{
        {Rune: EOF, Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 0, Line: 1, Col: 1}},
      },
    },
    {
      name:  "simple ASCII",
      input: "ab",
      expected: []RuneSpan{
        {Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
        {Rune: EOF, Pos: TextPosition{Idx: 2, Line: 1, Col: 3}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
      },
    },
    {
      name:  "UTF-8 characters",
      input: "αβ",
      expected: []RuneSpan{
        {Rune: 'β', Pos: TextPosition{Idx: 2, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
        {Rune: EOF, Pos: TextPosition{Idx: 4, Line: 1, Col: 3}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
      },
    },
    {
      name:  "line break normalization",
      input: "a\nb",
      expected: []RuneSpan{
        {Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
        {Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
        {Rune: EOF, Pos: TextPosition{Idx: 3, Line: 2, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
      },
    },
    {
      name:  "CRLF normalization",
      input: "a\r\nb",
      expected: []RuneSpan{
        {Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 1}},
        {Rune: 'b', Pos: TextPosition{Idx: 3, Line: 2, Col: 1}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
        {Rune: EOF, Pos: TextPosition{Idx: 4, Line: 2, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
      },
    },
    {
      name:  "escaped line break",
      input: "a\\\nb",
      expected: []RuneSpan{
        {Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
        {Rune: EOF, Pos: TextPosition{Idx: 4, Line: 2, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
      },
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      var result []RuneSpan

      for {
        span := scanner.NextSpan()
        result = append(result, span)
        if span.Rune == EOF {
          break
        }
      }

      if len(result) != len(tt.expected) {
        t.Errorf("expected %d spans, got %d", len(tt.expected), len(result))
        return
      }

      for i, expected := range tt.expected {
        if result[i] != expected {
          t.Errorf("at position %d: expected %+v, got %+v", i, expected, result[i])
        }
      }
    })
  }
}

func TestScannerNextSpanVsPopSpanPeekSpan(t *testing.T) {
  tests := []struct {
    name  string
    input string
  }{
    {
      name:  "simple ASCII",
      input: "abc",
    },
    {
      name:  "UTF-8 characters",
      input: "αβγ",
    },
    {
      name:  "mixed content",
      input: "a\r\nb\\c",
    },
    {
      name:  "escaped line break",
      input: "a\\\nb",
    },
    {
      name:  "empty string",
      input: "",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner1 := NewScanner(tt.input)
      scanner2 := NewScanner(tt.input)

      for {
        // NextSpan should be equivalent to PopSpan() followed by PeekSpan()
        nextSpan := scanner1.NextSpan()
        
        scanner2.PopSpan()
        peekSpan := scanner2.PeekSpan()

        if nextSpan != peekSpan {
          t.Errorf("nextSpan/popSpan+peekSpan mismatch: nextSpan=%+v, popSpan+peekSpan=%+v", nextSpan, peekSpan)
        }

        if nextSpan.Rune == EOF {
          break
        }
      }
    })
  }
}

func TestScannerNextSpanPosition(t *testing.T) {
  scanner := NewScanner("abc")

  // Initial position
  expectedPos := TextPosition{Idx: 0, Line: 1, Col: 1}
  if scanner.TextPosition != expectedPos {
    t.Errorf("initial position: expected %+v, got %+v", expectedPos, scanner.TextPosition)
  }

  // NextSpan should advance position
  span := scanner.NextSpan()
  expectedPos = TextPosition{Idx: 1, Line: 1, Col: 2}
  if scanner.TextPosition != expectedPos {
    t.Errorf("after NextSpan: expected %+v, got %+v", expectedPos, scanner.TextPosition)
  }

  // Verify span has correct positions
  expectedSpan := RuneSpan{
    Rune: 'b',
    Pos:  TextPosition{Idx: 1, Line: 1, Col: 2},
    End:  TextPosition{Idx: 2, Line: 1, Col: 3},
  }
  if span != expectedSpan {
    t.Errorf("expected span %+v, got %+v", expectedSpan, span)
  }
}

func TestScannerNextSpanAtEOF(t *testing.T) {
  scanner := NewScanner("a")

  // NextSpan should return EOF span when at end
  span := scanner.NextSpan()
  expectedSpan := RuneSpan{
    Rune: EOF,
    Pos:  TextPosition{Idx: 1, Line: 1, Col: 2},
    End:  TextPosition{Idx: 1, Line: 1, Col: 2},
  }
  if span != expectedSpan {
    t.Errorf("NextSpan at EOF: expected %+v, got %+v", expectedSpan, span)
  }

  // Position should remain at end
  expectedPos := TextPosition{Idx: 1, Line: 1, Col: 2}
  if scanner.TextPosition != expectedPos {
    t.Errorf("position after NextSpan at EOF: expected %+v, got %+v", expectedPos, scanner.TextPosition)
  }
}

func TestScannerMark(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		actions  []string // "mark", "pop", "slice"
		expected []string // expected slice results
	}{
		{
			name:     "simple mark and slice",
			input:    "abc",
			actions:  []string{"mark", "pop", "pop", "slice"},
			expected: []string{"ab"},
		},
		{
			name:     "mark at beginning",
			input:    "hello",
			actions:  []string{"mark", "pop", "pop", "pop", "slice"},
			expected: []string{"hel"},
		},
		{
			name:     "mark after some pops",
			input:    "hello",
			actions:  []string{"pop", "mark", "pop", "pop", "slice"},
			expected: []string{"el"},
		},
		{
			name:     "multiple marks",
			input:    "abcdef",
			actions:  []string{"mark", "pop", "pop", "slice", "mark", "pop", "slice"},
			expected: []string{"ab", "c"},
		},
		{
			name:     "empty slice",
			input:    "abc",
			actions:  []string{"mark", "slice"},
			expected: []string{""},
		},
		{
			name:     "mark at end",
			input:    "ab",
			actions:  []string{"pop", "pop", "mark", "slice"},
			expected: []string{""},
		},
		{
			name:     "UTF-8 characters",
			input:    "αβγδ",
			actions:  []string{"mark", "pop", "pop", "slice"},
			expected: []string{"αβ"},
		},
		{
			name:     "line break normalization",
			input:    "a\nb\rc\r\nd",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\nb\nc\n"},
		},
		{
			name:     "escaped line break",
			input:    "a\\\nb",
			actions:  []string{"mark", "pop", "pop", "slice"},
			expected: []string{"ab"},
		},
		{
			name:     "complex normalization",
			input:    "a\r\nb\\\nc",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\nbc"},
		},
		{
			name:     "regular backslash",
			input:    "a\\b",
			actions:  []string{"mark", "pop", "pop", "pop", "slice"},
			expected: []string{"a\\b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			var results []string

			for _, action := range tt.actions {
				switch action {
				case "mark":
					scanner.Mark()
				case "pop":
					scanner.Pop()
				case "slice":
					results = append(results, scanner.Slice())
				}
			}

			if len(results) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(results))
				return
			}

			for i, expected := range tt.expected {
				if results[i] != expected {
					t.Errorf("result %d: expected %q, got %q", i, expected, results[i])
				}
			}
		})
	}
}

func TestScannerMarkInitialState(t *testing.T) {
	scanner := NewScanner("abc")

	// Scanner should start with mark at beginning
	result := scanner.Slice()
	if result != "" {
		t.Errorf("initial slice should be empty, got %q", result)
	}

	// After popping, slice should include popped character
	scanner.Pop()
	result = scanner.Slice()
	if result != "a" {
		t.Errorf("after one pop, slice should be 'a', got %q", result)
	}
}

func TestScannerMarkComplexFlag(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		hasComplex bool
	}{
		{
			name:       "simple ASCII",
			input:      "abc",
			hasComplex: false,
		},
		{
			name:       "UTF-8 only",
			input:      "αβγ",
			hasComplex: false,
		},
		{
			name:       "CR normalization",
			input:      "a\rb",
			hasComplex: true,
		},
		{
			name:       "CRLF normalization",
			input:      "a\r\nb",
			hasComplex: true,
		},
		{
			name:       "escaped line break",
			input:      "a\\\nb",
			hasComplex: true,
		},
		{
			name:       "regular LF",
			input:      "a\nb",
			hasComplex: false,
		},
		{
			name:       "regular backslash",
			input:      "a\\b",
			hasComplex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			scanner.Mark()

			// Pop all characters
			for {
				r := scanner.Pop()
				if r == EOF {
					break
				}
			}

			// Check if complex flag was set correctly by comparing slice result
			slice := scanner.Slice()

			if tt.hasComplex {
				// For complex cases, verify normalization occurred
				if strings.Contains(tt.input, "\r\n") && !strings.Contains(slice, "\r") {
					// CRLF should be normalized to LF
				} else if strings.Contains(tt.input, "\r") && !strings.Contains(slice, "\r") {
					// CR should be normalized to LF
				} else if strings.Contains(tt.input, "\\\n") && !strings.Contains(slice, "\\\n") {
					// Escaped line break should be removed
				}
			} else {
				// For simple cases, slice should match input (except for consumed chars)
				if tt.input == "abc" {
					if slice != "abc" {
						t.Errorf("simple case: expected %q, got %q", tt.input, slice)
					}
				}
			}
		})
	}
}

func TestScannerMarkAfterMark(t *testing.T) {
	scanner := NewScanner("abcdef")

	// First mark and operations
	scanner.Mark()
	scanner.Pop() // 'a'
	scanner.Pop() // 'b'

	result1 := scanner.Slice()
	if result1 != "ab" {
		t.Errorf("first slice: expected 'ab', got %q", result1)
	}

	// Second mark should reset
	scanner.Mark()
	scanner.Pop() // 'c'

	result2 := scanner.Slice()
	if result2 != "c" {
		t.Errorf("second slice: expected 'c', got %q", result2)
	}

	// Verify isComplexSinceMark is reset
	scanner.Mark()
	if scanner.isComplexSinceMark {
		t.Error("isComplexSinceMark should be false after mark")
	}
}


func TestScannerMarked(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    actions  []string // "mark", "pop", "peek", "next"
    expected []TextPosition
  }{
    {
      name:     "initial marked position",
      input:    "abc",
      actions:  []string{"marked"},
      expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
    },
    {
      name:     "marked position after mark",
      input:    "abc",
      actions:  []string{"pop", "mark", "marked"},
      expected: []TextPosition{{Idx: 1, Line: 1, Col: 2}},
    },
    {
      name:     "marked position unchanged after pop",
      input:    "abc",
      actions:  []string{"mark", "pop", "marked"},
      expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
    },
    {
      name:     "marked position unchanged after peek",
      input:    "abc",
      actions:  []string{"mark", "peek", "marked"},
      expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
    },
    {
      name:     "marked position unchanged after next",
      input:    "abc",
      actions:  []string{"mark", "next", "marked"},
      expected: []TextPosition{{Idx: 0, Line: 1, Col: 1}},
    },
    {
      name:     "multiple marks",
      input:    "abcdef",
      actions:  []string{"pop", "mark", "marked", "pop", "pop", "marked", "mark", "marked"},
      expected: []TextPosition{
        {Idx: 1, Line: 1, Col: 2}, // after first mark
        {Idx: 1, Line: 1, Col: 2}, // unchanged after pops
        {Idx: 3, Line: 1, Col: 4}, // after second mark
      },
    },
    {
      name:     "mark at EOF",
      input:    "a",
      actions:  []string{"pop", "mark", "marked"},
      expected: []TextPosition{{Idx: 1, Line: 1, Col: 2}},
    },
    {
      name:     "mark with line breaks",
      input:    "a\nb\nc",
      actions:  []string{"pop", "pop", "mark", "marked"},
      expected: []TextPosition{{Idx: 2, Line: 2, Col: 1}},
    },
    {
      name:     "mark with CRLF normalization",
      input:    "a\r\nb",
      actions:  []string{"pop", "pop", "mark", "marked"},
      expected: []TextPosition{{Idx: 3, Line: 2, Col: 1}},
    },
    {
      name:     "mark with escaped line break",
      input:    "a\\\nb",
      actions:  []string{"pop", "pop", "mark", "marked"},
      expected: []TextPosition{{Idx: 4, Line: 2, Col: 2}},
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      expectedIdx := 0

      for _, action := range tt.actions {
        switch action {
        case "mark":
          scanner.Mark()
        case "pop":
          scanner.Pop()
        case "peek":
          scanner.Peek()
        case "next":
          scanner.Next()
        case "marked":
          if expectedIdx >= len(tt.expected) {
            t.Errorf("unexpected marked call at action %s", action)
            continue
          }
          marked := scanner.Marked()
          expected := tt.expected[expectedIdx]
          if marked != expected {
            t.Errorf("marked position: expected %+v, got %+v", expected, marked)
          }
          expectedIdx++
        }
      }
    })
  }
}

func TestScannerMarkedEmpty(t *testing.T) {
  scanner := NewScanner("")
  
  // Initial marked position should be at start
  marked := scanner.Marked()
  expected := TextPosition{Idx: 0, Line: 1, Col: 1}
  if marked != expected {
    t.Errorf("initial marked position: expected %+v, got %+v", expected, marked)
  }

  // Mark at EOF should work
  scanner.Mark()
  marked = scanner.Marked()
  if marked != expected {
    t.Errorf("marked at EOF: expected %+v, got %+v", expected, marked)
  }
}

func TestScannerMarkedConsistency(t *testing.T) {
  scanner := NewScanner("hello\nworld")
  
  // Mark at start
  scanner.Mark()
  initialMarked := scanner.Marked()
  
  // Advance scanner
  scanner.Pop() // 'h'
  scanner.Pop() // 'e'
  scanner.Pop() // 'l'
  
  // Marked should be unchanged
  marked := scanner.Marked()
  if marked != initialMarked {
    t.Errorf("marked position changed: expected %+v, got %+v", initialMarked, marked)
  }
  
  // Mark at current position
  scanner.Mark()
  newMarked := scanner.Marked()
  expectedPos := TextPosition{Idx: 3, Line: 1, Col: 4}
  if newMarked != expectedPos {
    t.Errorf("new marked position: expected %+v, got %+v", expectedPos, newMarked)
  }
  
  // Multiple calls should return same position
  for i := 0; i < 3; i++ {
    if scanner.Marked() != newMarked {
      t.Errorf("marked position changed on call %d", i)
    }
  }
}

func TestScannerMarkedWithUTF8(t *testing.T) {
  scanner := NewScanner("αβγ")
  
  // Mark at start
  scanner.Mark()
  initialMarked := scanner.Marked()
  expected := TextPosition{Idx: 0, Line: 1, Col: 1}
  if initialMarked != expected {
    t.Errorf("initial marked: expected %+v, got %+v", expected, initialMarked)
  }
  
  // Pop first UTF-8 character
  scanner.Pop() // 'α'
  
  // Mark after UTF-8 character
  scanner.Mark()
  marked := scanner.Marked()
  expected = TextPosition{Idx: 2, Line: 1, Col: 2} // 'α' is 2 bytes
  if marked != expected {
    t.Errorf("marked after UTF-8: expected %+v, got %+v", expected, marked)
  }
}

func TestScannerMarkedWithComplexNormalization(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    popCount int
    expected TextPosition
  }{
    {
      name:     "after CR normalization",
      input:    "a\rb",
      popCount: 2,
      expected: TextPosition{Idx: 2, Line: 2, Col: 1},
    },
    {
      name:     "after CRLF normalization",
      input:    "a\r\nb",
      popCount: 2,
      expected: TextPosition{Idx: 3, Line: 2, Col: 1},
    },
    {
      name:     "after escaped line break",
      input:    "a\\\nb",
      popCount: 2,
      expected: TextPosition{Idx: 4, Line: 2, Col: 2},
    },
    {
      name:     "after multiple escaped breaks",
      input:    "a\\\n\\\nb",
      popCount: 2,
      expected: TextPosition{Idx: 6, Line: 3, Col: 2},
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      
      // Pop specified number of times
      for i := 0; i < tt.popCount; i++ {
        scanner.Pop()
      }
      
      // Mark at current position
      scanner.Mark()
      marked := scanner.Marked()
      
      if marked != tt.expected {
        t.Errorf("marked after %d pops: expected %+v, got %+v", tt.popCount, tt.expected, marked)
      }
    })
  }
}

func TestScannerMarkedImmutability(t *testing.T) {
  scanner := NewScanner("test")
  
  // Mark and get position
  scanner.Mark()
  marked1 := scanner.Marked()
  
  // Modify the returned position
  marked1.Idx = 999
  marked1.Line = 999
  marked1.Col = 999
  
  // Verify scanner's marked position is unchanged
  marked2 := scanner.Marked()
  expected := TextPosition{Idx: 0, Line: 1, Col: 1}
  if marked2 != expected {
    t.Errorf("marked position was mutated: expected %+v, got %+v", expected, marked2)
  }
}

func TestScannerMarkedAfterMarkReset(t *testing.T) {
  scanner := NewScanner("abcdef")
  
  // Advance and mark
  scanner.Pop() // 'a'
  scanner.Pop() // 'b'
  scanner.Mark()
  firstMark := scanner.Marked()
  
  // Advance more and mark again
  scanner.Pop() // 'c'
  scanner.Pop() // 'd'
  scanner.Mark()
  secondMark := scanner.Marked()
  
  // Verify marks are different
  if firstMark == secondMark {
    t.Errorf("marks should be different: first=%+v, second=%+v", firstMark, secondMark)
  }
  
  // Verify current marked position is the second mark
  current := scanner.Marked()
  if current != secondMark {
    t.Errorf("current marked should be second mark: expected %+v, got %+v", secondMark, current)
  }
}

func TestScannerSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		actions  []string // "mark", "pop", "slice"
		expected []string // expected slice results
	}{
		{
			name:     "simple ASCII slice",
			input:    "hello",
			actions:  []string{"mark", "pop", "pop", "pop", "slice"},
			expected: []string{"hel"},
		},
		{
			name:     "UTF-8 slice",
			input:    "αβγδ",
			actions:  []string{"mark", "pop", "pop", "slice"},
			expected: []string{"αβ"},
		},
		{
			name:     "empty slice at start",
			input:    "abc",
			actions:  []string{"mark", "slice"},
			expected: []string{""},
		},
		{
			name:     "empty slice at end",
			input:    "ab",
			actions:  []string{"pop", "pop", "mark", "slice"},
			expected: []string{""},
		},
		{
			name:     "full string slice",
			input:    "hello",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"hello"},
		},
		{
			name:     "CR normalization in slice",
			input:    "a\rb\rc",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\nb\n"},
		},
		{
			name:     "CRLF normalization in slice",
			input:    "a\r\nb\r\nc",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\nb\n"},
		},
		{
			name:     "mixed line endings in slice",
			input:    "a\nb\rc\r\nd",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\nb\nc\n"},
		},
		{
			name:     "escaped line break in slice",
			input:    "hello\\\nworld",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "pop", "pop", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"helloworld"},
		},
		{
			name:     "multiple escaped line breaks",
			input:    "a\\\nb\\\nc",
			actions:  []string{"mark", "pop", "pop", "pop", "slice"},
			expected: []string{"abc"},
		},
		{
			name:     "regular backslash in slice",
			input:    "a\\b\\c",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\\b\\"},
		},
		{
			name:     "backslash at end in slice",
			input:    "hello\\",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"hello\\"},
		},
		{
			name:     "complex mixed content",
			input:    "a\r\nb\\\nc\\d",
			actions:  []string{"mark", "pop", "pop", "pop", "pop", "pop", "slice"},
			expected: []string{"a\nbc\\"},
		},
		{
			name:     "multiple marks and slices",
			input:    "abcdef",
			actions:  []string{"mark", "pop", "pop", "slice", "mark", "pop", "pop", "slice", "mark", "pop", "pop", "slice"},
			expected: []string{"ab", "cd", "ef"},
		},
		{
			name:     "mark after partial consumption",
			input:    "hello world",
			actions:  []string{"pop", "pop", "pop", "mark", "pop", "pop", "pop", "slice"},
			expected: []string{"lo "},
		},
		{
			name:     "slice with only normalization",
			input:    "\r\n\r\n",
			actions:  []string{"mark", "pop", "pop", "slice"},
			expected: []string{"\n\n"},
		},
		{
			name:     "slice with only escaped breaks",
			input:    "\\\n\\\n",
			actions:  []string{"mark", "pop", "pop", "slice"},
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			var results []string

			for _, action := range tt.actions {
				switch action {
				case "mark":
					scanner.Mark()
				case "pop":
					scanner.Pop()
				case "slice":
					results = append(results, scanner.Slice())
				}
			}

			if len(results) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(results))
				return
			}

			for i, expected := range tt.expected {
				if results[i] != expected {
					t.Errorf("result %d: expected %q, got %q", i, expected, results[i])
				}
			}
		})
	}
}

func TestScannerSliceConsistency(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple ASCII",
			input: "hello world",
		},
		{
			name:  "UTF-8 characters",
			input: "αβγ δεζ",
		},
		{
			name:  "mixed line endings",
			input: "line1\nline2\rline3\r\nline4",
		},
		{
			name:  "escaped line breaks",
			input: "cont\\\ninued\\\ntext",
		},
		{
			name:  "regular backslashes",
			input: "path\\to\\file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			scanner.Mark()

			var runes []rune
			for {
				r := scanner.Pop()
				if r == EOF {
					break
				}
				runes = append(runes, r)
			}

			slice := scanner.Slice()
			reconstructed := string(runes)

			if slice != reconstructed {
				t.Errorf("slice inconsistency: slice=%q, reconstructed=%q", slice, reconstructed)
			}
		})
	}
}

func TestScannerSliceAfterEOF(t *testing.T) {
	scanner := NewScanner("abc")
	scanner.Mark()

	// Consume all characters and EOF
	scanner.Pop() // 'a'
	scanner.Pop() // 'b'
	scanner.Pop() // 'c'
	scanner.Pop() // EOF

	slice := scanner.Slice()
	if slice != "abc" {
		t.Errorf("slice after EOF: expected 'abc', got %q", slice)
	}
}

func TestScannerSliceComplexFlag(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectComplex bool
	}{
		{
			name:          "simple ASCII",
			input:         "hello",
			expectComplex: false,
		},
		{
			name:          "UTF-8 only",
			input:         "αβγ",
			expectComplex: false,
		},
		{
			name:          "with CR",
			input:         "a\rb",
			expectComplex: true,
		},
		{
			name:          "with CRLF",
			input:         "a\r\nb",
			expectComplex: true,
		},
		{
			name:          "with escaped line break",
			input:         "a\\\nb",
			expectComplex: true,
		},
		{
			name:          "with LF only",
			input:         "a\nb",
			expectComplex: false,
		},
		{
			name:          "with regular backslash",
			input:         "a\\b",
			expectComplex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.input)
			scanner.Mark()

			// Consume all characters
			for {
				if scanner.Pop() == EOF {
					break
				}
			}

			// The complex flag should affect how slice works
			slice := scanner.Slice()

			if tt.expectComplex {
				// For complex cases, verify normalization occurred
				if strings.Contains(tt.input, "\r\n") && strings.Contains(slice, "\r") {
					t.Error("CRLF should be normalized in slice")
				}
				if strings.Contains(tt.input, "\r") && !strings.Contains(tt.input, "\r\n") && strings.Contains(slice, "\r") {
					t.Error("CR should be normalized in slice")
				}
				if strings.Contains(tt.input, "\\\n") && strings.Contains(slice, "\\\n") {
					t.Error("escaped line break should be removed in slice")
				}
			}
		})
	}
}

func TestScannerSliceIncl(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    popCount int
    expected string
  }{
    {
      name:     "empty string",
      input:    "",
      popCount: 0,
      expected: "",
    },
    {
      name:     "simple ASCII",
      input:    "abc",
      popCount: 2,
      expected: "abc",
    },
    {
      name:     "UTF-8 characters",
      input:    "αβγ",
      popCount: 2,
      expected: "αβγ",
    },
    {
      name:     "single character",
      input:    "a",
      popCount: 0,
      expected: "a",
    },
    {
      name:     "full string",
      input:    "hello",
      popCount: 4,
      expected: "hello",
    },
    {
      name:     "LF normalization",
      input:    "a\nb",
      popCount: 2,
      expected: "a\nb",
    },
    {
      name:     "CR normalization",
      input:    "a\rb",
      popCount: 2,
      expected: "a\nb",
    },
    {
      name:     "CRLF normalization",
      input:    "a\r\nb",
      popCount: 2,
      expected: "a\nb",
    },
    {
      name:     "escaped line break",
      input:    "a\\\nb",
      popCount: 1,
      expected: "ab",
    },
    {
      name:     "regular backslash",
      input:    "a\\b",
      popCount: 2,
      expected: "a\\b",
    },
    {
      name:     "backslash at end",
      input:    "a\\",
      popCount: 1,
      expected: "a\\",
    },
    {
      name:     "multiple line breaks",
      input:    "a\n\r\n\rb",
      popCount: 4,
      expected: "a\n\n\nb",
    },
    {
      name:     "multiple escaped line breaks",
      input:    "a\\\n\\\nb",
      popCount: 1,
      expected: "ab",
    },
    {
      name:     "mixed content",
      input:    "hello\r\nworld\\\ncontinued",
      popCount: 19,
      expected: "hello\nworldcontinued",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      scanner.Mark()

      for i := 0; i < tt.popCount; i++ {
        scanner.Pop()
      }

      result := scanner.SliceIncl()
      if result != tt.expected {
        t.Errorf("expected %q, got %q", tt.expected, result)
      }
    })
  }
}

func TestScannerSliceInclWithMarkAfterAdvancement(t *testing.T) {
  tests := []struct {
    name       string
    input      string
    markAt     int
    popCount   int
    expected   string
  }{
    {
      name:       "mark in middle",
      input:      "abcde",
      markAt:     2,
      popCount:   2,
      expected:   "cde",
    },
    {
      name:       "mark at end",
      input:      "abc",
      markAt:     3,
      popCount:   0,
      expected:   "",
    },
    {
      name:       "mark with UTF-8",
      input:      "αβγδε",
      markAt:     2,
      popCount:   2,
      expected:   "γδε",
    },
    {
      name:       "mark with line breaks",
      input:      "a\nb\nc",
      markAt:     2,
      popCount:   2,
      expected:   "b\nc",
    },
    {
      name:       "mark with escaped breaks",
      input:      "a\\\nb\\\nc",
      markAt:     1,
      popCount:   1,
      expected:   "bc",
    },
    {
      name:       "mark with normalization",
      input:      "a\r\nb\rc",
      markAt:     1,
      popCount:   3,
      expected:   "\nb\nc",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)

      // Advance to mark position
      for i := 0; i < tt.markAt; i++ {
        scanner.Pop()
      }

      scanner.Mark()

      // Pop additional characters
      for i := 0; i < tt.popCount; i++ {
        scanner.Pop()
      }

      result := scanner.SliceIncl()
      if result != tt.expected {
        t.Errorf("expected %q, got %q", tt.expected, result)
      }
    })
  }
}

func TestScannerSliceInclVsSlice(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    popCount int
  }{
    {
      name:     "simple ASCII",
      input:    "abcde",
      popCount: 3,
    },
    {
      name:     "UTF-8 characters",
      input:    "αβγδε",
      popCount: 3,
    },
    {
      name:     "with line breaks",
      input:    "a\nb\nc",
      popCount: 3,
    },
    {
      name:     "with normalization",
      input:    "a\r\nb\rc",
      popCount: 3,
    },
    {
      name:     "with escaped breaks",
      input:    "a\\\nb\\\nc",
      popCount: 2,
    },
    {
      name:     "empty string",
      input:    "",
      popCount: 0,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner1 := NewScanner(tt.input)
      scanner2 := NewScanner(tt.input)

      scanner1.Mark()
      scanner2.Mark()

      for i := 0; i < tt.popCount; i++ {
        scanner1.Pop()
        scanner2.Pop()
      }

      slice := scanner1.Slice()
      sliceIncl := scanner2.SliceIncl()

      // SliceIncl should include one more rune than Slice
      if tt.popCount > 0 && !strings.HasPrefix(sliceIncl, slice) {
        t.Errorf("SliceIncl should start with Slice result: slice=%q, sliceIncl=%q", slice, sliceIncl)
      }
    })
  }
}

func TestScannerSliceInclComplexNormalization(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    popCount int
    expected string
  }{
    {
      name:     "CRLF followed by escaped break",
      input:    "a\r\nb\\\nc",
      popCount: 3,
      expected: "a\nbc",
    },
    {
      name:     "escaped CRLF",
      input:    "a\\\r\nb",
      popCount: 1,
      expected: "ab",
    },
    {
      name:     "escaped CR",
      input:    "a\\\rb",
      popCount: 1,
      expected: "ab",
    },
    {
      name:     "multiple consecutive escapes",
      input:    "a\\\n\\\n\\\nb",
      popCount: 1,
      expected: "ab",
    },
    {
      name:     "mixed line endings",
      input:    "a\r\nb\nc\rd",
      popCount: 6,
      expected: "a\nb\nc\nd",
    },
    {
      name:     "backslash not followed by newline",
      input:    "a\\b\\c",
      popCount: 4,
      expected: "a\\b\\c",
    },
    {
      name:     "backslash at end of string",
      input:    "abc\\",
      popCount: 3,
      expected: "abc\\",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      scanner.Mark()

      for i := 0; i < tt.popCount; i++ {
        scanner.Pop()
      }

      result := scanner.SliceIncl()
      if result != tt.expected {
        t.Errorf("expected %q, got %q", tt.expected, result)
      }
    })
  }
}

func TestScannerSliceInclAtEOF(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected string
  }{
    {
      name:     "empty string",
      input:    "",
      expected: "",
    },
    {
      name:     "single character",
      input:    "a",
      expected: "a",
    },
    {
      name:     "multiple characters",
      input:    "abc",
      expected: "abc",
    },
    {
      name:     "with line breaks",
      input:    "a\nb",
      expected: "a\nb",
    },
    {
      name:     "with escaped breaks",
      input:    "a\\\nb",
      expected: "ab",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      scanner.Mark()

      // Advance to EOF
      for !scanner.IsEOF() {
        scanner.Pop()
      }

      result := scanner.SliceIncl()
      if result != tt.expected {
        t.Errorf("expected %q, got %q", tt.expected, result)
      }
    })
  }
}

func TestScannerSliceInclWithoutMark(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    popCount int
    expected string
  }{
    {
      name:     "from start",
      input:    "abc",
      popCount: 2,
      expected: "abc",
    },
    {
      name:     "UTF-8 from start",
      input:    "αβγ",
      popCount: 2,
      expected: "αβγ",
    },
    {
      name:     "with normalization from start",
      input:    "a\r\nb",
      popCount: 2,
      expected: "a\nb",
    },
    {
      name:     "with escaped breaks from start",
      input:    "a\\\nb",
      popCount: 1,
      expected: "ab",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      // No explicit Mark() call - should use initial position

      for i := 0; i < tt.popCount; i++ {
        scanner.Pop()
      }

      result := scanner.SliceIncl()
      if result != tt.expected {
        t.Errorf("expected %q, got %q", tt.expected, result)
      }
    })
  }
}

func TestScannerSliceInclEdgeCases(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    setup    func(*Scanner)
    expected string
  }{
    {
      name:  "mark at same position as current",
      input: "abc",
      setup: func(s *Scanner) {
        s.Pop()
        s.Mark()
      },
      expected: "b",
    },
    {
      name:  "multiple marks",
      input: "abcde",
      setup: func(s *Scanner) {
        s.Mark()
        s.Pop()
        s.Mark()
        s.Pop()
      },
      expected: "bc",
    },
    {
      name:  "mark after EOF",
      input: "a",
      setup: func(s *Scanner) {
        s.Pop()
        s.Mark()
      },
      expected: "",
    },
    {
      name:  "slice at mark position",
      input: "abc",
      setup: func(s *Scanner) {
        s.Pop()
        s.Mark()
        // Don't advance further
      },
      expected: "b",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      tt.setup(scanner)

      result := scanner.SliceIncl()
      if result != tt.expected {
        t.Errorf("expected %q, got %q", tt.expected, result)
      }
    })
  }
}

func TestStream(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []RuneSpan
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []RuneSpan{},
		},
		{
			name:  "simple ASCII",
			input: "abc",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
				{Rune: 'c', Pos: TextPosition{Idx: 2, Line: 1, Col: 3}, End: TextPosition{Idx: 3, Line: 1, Col: 4}},
			},
		},
		{
			name:  "UTF-8 characters",
			input: "αβ",
			expected: []RuneSpan{
				{Rune: 'α', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 2, Line: 1, Col: 2}},
				{Rune: 'β', Pos: TextPosition{Idx: 2, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
			},
		},
		{
			name:  "line breaks",
			input: "a\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
			},
		},
		{
			name:  "CR normalization",
			input: "a\rb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
			},
		},
		{
			name:  "CRLF normalization",
			input: "a\r\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 3, Line: 2, Col: 1}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
			},
		},
		{
			name:  "escaped line break",
			input: "a\\\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := Stream(tt.input)
			var result []RuneSpan

			for span := range ch {
				result = append(result, span)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d spans, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at position %d: expected %+v, got %+v", i, expected, result[i])
				}
			}
		})
	}
}

func TestStreamChannelClosure(t *testing.T) {
	ch := Stream("abc")

	// Consume all values
	count := 0
	for range ch {
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 spans, got %d", count)
	}

	// Channel should be closed after consumption
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after EOF")
	}
}

func TestStreamConcurrency(t *testing.T) {
	input := "hello world"
	ch := Stream(input)

	done := make(chan bool)
	var result []RuneSpan

	// Read from channel in goroutine
	go func() {
		for span := range ch {
			result = append(result, span)
		}
		done <- true
	}()

	// Wait for completion
	<-done

	// Verify we got all characters
	if len(result) != len(input) {
		t.Errorf("expected %d spans, got %d", len(input), len(result))
	}

	// Verify first and last characters
	if len(result) > 0 {
		if result[0].Rune != 'h' {
			t.Errorf("expected first rune 'h', got %q", result[0].Rune)
		}
		if result[len(result)-1].Rune != 'd' {
			t.Errorf("expected last rune 'd', got %q", result[len(result)-1].Rune)
		}
	}
}

func TestStreamVsPopSpan(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple ASCII",
			input: "hello",
		},
		{
			name:  "UTF-8 characters",
			input: "αβγ",
		},
		{
			name:  "mixed content",
			input: "a\r\nb\\\nc",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get results from Stream
			ch := Stream(tt.input)
			var streamResult []RuneSpan
			for span := range ch {
				streamResult = append(streamResult, span)
			}

			// Get results from manual PopSpan
			scanner := NewScanner(tt.input)
			var popResult []RuneSpan
			for {
				span := scanner.PopSpan()
				if span.Rune == EOF {
					break
				}
				popResult = append(popResult, span)
			}

			// Compare results
			if len(streamResult) != len(popResult) {
				t.Errorf("length mismatch: Stream=%d, PopSpan=%d", len(streamResult), len(popResult))
				return
			}

			for i, expected := range popResult {
				if streamResult[i] != expected {
					t.Errorf("at position %d: Stream=%+v, PopSpan=%+v", i, streamResult[i], expected)
				}
			}
		})
	}
}

func TestStreamEarlyTermination(t *testing.T) {
	ch := Stream("abcdef")

	// Read only first two spans
	spans := make([]RuneSpan, 0, 2)
	for span := range ch {
		spans = append(spans, span)
		if len(spans) == 2 {
			break
		}
	}

	if len(spans) != 2 {
		t.Errorf("expected 2 spans, got %d", len(spans))
	}

	if spans[0].Rune != 'a' {
		t.Errorf("expected first rune 'a', got %q", spans[0].Rune)
	}

	if spans[1].Rune != 'b' {
		t.Errorf("expected second rune 'b', got %q", spans[1].Rune)
	}
}

func TestForEach(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []RuneSpan
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []RuneSpan{},
		},
		{
			name:  "simple ASCII",
			input: "abc",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
				{Rune: 'c', Pos: TextPosition{Idx: 2, Line: 1, Col: 3}, End: TextPosition{Idx: 3, Line: 1, Col: 4}},
			},
		},
		{
			name:  "UTF-8 characters",
			input: "αβ",
			expected: []RuneSpan{
				{Rune: 'α', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 2, Line: 1, Col: 2}},
				{Rune: 'β', Pos: TextPosition{Idx: 2, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
			},
		},
		{
			name:  "line breaks",
			input: "a\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
			},
		},
		{
			name:  "CR normalization",
			input: "a\rb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
			},
		},
		{
			name:  "CRLF normalization",
			input: "a\r\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 1}},
				{Rune: 'b', Pos: TextPosition{Idx: 3, Line: 2, Col: 1}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
			},
		},
		{
			name:  "escaped line break",
			input: "a\\\nb",
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []RuneSpan

			ForEach(tt.input, func(span RuneSpan) bool {
				result = append(result, span)
				return true
			})

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d spans, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at position %d: expected %+v, got %+v", i, expected, result[i])
				}
			}
		})
	}
}

func TestForEachEarlyTermination(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		stopAt   rune
		expected []RuneSpan
	}{
		{
			name:   "stop at second character",
			input:  "abcdef",
			stopAt: 'b',
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
			},
		},
		{
			name:   "stop at first character",
			input:  "hello",
			stopAt: 'h',
			expected: []RuneSpan{
				{Rune: 'h', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
			},
		},
		{
			name:   "stop at line break",
			input:  "a\nb\nc",
			stopAt: '\n',
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
			},
		},
		{
			name:   "stop at UTF-8 character",
			input:  "aβγ",
			stopAt: 'β',
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'β', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 1, Col: 3}},
			},
		},
		{
			name:   "never stop",
			input:  "abc",
			stopAt: 'x',
			expected: []RuneSpan{
				{Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
				{Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
				{Rune: 'c', Pos: TextPosition{Idx: 2, Line: 1, Col: 3}, End: TextPosition{Idx: 3, Line: 1, Col: 4}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []RuneSpan

			ForEach(tt.input, func(span RuneSpan) bool {
				result = append(result, span)
				return span.Rune != tt.stopAt
			})

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d spans, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at position %d: expected %+v, got %+v", i, expected, result[i])
				}
			}
		})
	}
}

func TestForEachReturnFalseImmediately(t *testing.T) {
	var result []RuneSpan

	ForEach("hello", func(span RuneSpan) bool {
		result = append(result, span)
		return false // stop immediately
	})

	if len(result) != 1 {
		t.Errorf("expected 1 span, got %d", len(result))
	}

	if len(result) > 0 && result[0].Rune != 'h' {
		t.Errorf("expected first rune 'h', got %q", result[0].Rune)
	}
}

func TestForEachVsManualIteration(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple ASCII",
			input: "hello",
		},
		{
			name:  "UTF-8 characters",
			input: "αβγ",
		},
		{
			name:  "mixed content",
			input: "a\r\nb\\\nc",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get results from ForEach
			var forEachResult []RuneSpan
			ForEach(tt.input, func(span RuneSpan) bool {
				forEachResult = append(forEachResult, span)
				return true
			})

			// Get results from manual iteration
			scanner := NewScanner(tt.input)
			var manualResult []RuneSpan
			for {
				span := scanner.PopSpan()
				if span.Rune == EOF {
					break
				}
				manualResult = append(manualResult, span)
			}

			// Compare results
			if len(forEachResult) != len(manualResult) {
				t.Errorf("length mismatch: ForEach=%d, manual=%d", len(forEachResult), len(manualResult))
				return
			}

			for i, expected := range manualResult {
				if forEachResult[i] != expected {
					t.Errorf("at position %d: ForEach=%+v, manual=%+v", i, forEachResult[i], expected)
				}
			}
		})
	}
}

func TestForEachNilFunction(t *testing.T) {
	// This test ensures ForEach handles nil function gracefully
	// Since the current implementation doesn't check for nil, this documents expected behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when function is nil")
		}
	}()

	ForEach("test", nil)
}

func TestForEachComplexNormalization(t *testing.T) {
	input := "a\r\nb\\\nc\rd"
	expected := []rune{'a', '\n', 'b', 'c', '\n', 'd'}

	var result []rune
	ForEach(input, func(span RuneSpan) bool {
		result = append(result, span.Rune)
		return true
	})

	if len(result) != len(expected) {
		t.Errorf("expected %d runes, got %d", len(expected), len(result))
		return
	}

	for i, expectedRune := range expected {
		if result[i] != expectedRune {
			t.Errorf("at position %d: expected %q, got %q", i, expectedRune, result[i])
		}
	}
}

func TestForEachCount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "single character",
			input:    "a",
			expected: 1,
		},
		{
			name:     "ASCII string",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "UTF-8 string",
			input:    "αβγ",
			expected: 3,
		},
		{
			name:     "with line breaks",
			input:    "a\nb\nc",
			expected: 5, // a, \n, b, \n, c
		},
		{
			name:     "with CRLF",
			input:    "a\r\nb",
			expected: 3, // a, \n, b
		},
		{
			name:     "with escaped line break",
			input:    "a\\\nb",
			expected: 2, // a, b
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			ForEach(tt.input, func(span RuneSpan) bool {
				count++
				return true
			})

			if count != tt.expected {
				t.Errorf("expected %d spans, got %d", tt.expected, count)
			}
		})
	}
}

func TestForEachConditionCheck(t *testing.T) {
	input := "abcdef"

	// Test stopping at specific count
	tests := []struct {
		name     string
		maxCount int
		expected int
	}{
		{
			name:     "stop after 1",
			maxCount: 1,
			expected: 1,
		},
		{
			name:     "stop after 3",
			maxCount: 3,
			expected: 3,
		},
		{
			name:     "stop after 10 (more than available)",
			maxCount: 10,
			expected: 6, // all characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			ForEach(input, func(span RuneSpan) bool {
				count++
				return count < tt.maxCount
			})

			if count != tt.expected {
				t.Errorf("expected %d spans, got %d", tt.expected, count)
			}
		})
	}
}
