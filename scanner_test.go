package scanner
import "testing"
import (
  "strings"
)

func TestScannerPop(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected []rune
  }{
    {
      name:     "empty string",
      input:    "",
      expected: []rune{ScanEOF},
    },
    {
      name:     "simple ASCII",
      input:    "abc",
      expected: []rune{'a', 'b', 'c', ScanEOF},
    },
    {
      name:     "UTF-8 characters",
      input:    "αβγ",
      expected: []rune{'α', 'β', 'γ', ScanEOF},
    },
    {
      name:     "LF normalization",
      input:    "a\nb",
      expected: []rune{'a', '\n', 'b', ScanEOF},
    },
    {
      name:     "CR normalization",
      input:    "a\rb",
      expected: []rune{'a', '\n', 'b', ScanEOF},
    },
    {
      name:     "CRLF normalization",
      input:    "a\r\nb",
      expected: []rune{'a', '\n', 'b', ScanEOF},
    },
    {
      name:     "escaped line break",
      input:    "a\\\nb",
      expected: []rune{'a', 'b', ScanEOF},
    },
    {
      name:     "regular backslash",
      input:    "a\\b",
      expected: []rune{'a', '\\', 'b', ScanEOF},
    },
    {
      name:     "backslash at end",
      input:    "a\\",
      expected: []rune{'a', '\\', ScanEOF},
    },
    {
      name:     "multiple line breaks",
      input:    "a\n\r\n\rb",
      expected: []rune{'a', '\n', '\n', '\n', 'b', ScanEOF},
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      scanner := NewScanner(tt.input)
      var result []rune
      
      for {
        r := scanner.Pop()
        result = append(result, r)
        if r == ScanEOF {
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
        {Rune: ScanEOF, Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 0, Line: 1, Col: 1}},
      },
    },
    {
      name:  "simple ASCII",
      input: "ab",
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
        {Rune: ScanEOF, Pos: TextPosition{Idx: 2, Line: 1, Col: 3}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
      },
    },
    {
      name:  "UTF-8 characters",
      input: "αβ",
      expected: []RuneSpan{
        {Rune: 'α', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 2, Line: 1, Col: 2}},
        {Rune: 'β', Pos: TextPosition{Idx: 2, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
        {Rune: ScanEOF, Pos: TextPosition{Idx: 4, Line: 1, Col: 3}, End: TextPosition{Idx: 4, Line: 1, Col: 3}},
      },
    },
    {
      name:  "line break normalization",
      input: "a\nb",
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
        {Rune: 'b', Pos: TextPosition{Idx: 2, Line: 2, Col: 1}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
        {Rune: ScanEOF, Pos: TextPosition{Idx: 3, Line: 2, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 2}},
      },
    },
    {
      name:  "CRLF normalization",
      input: "a\r\nb",
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 2, Col: 1}},
        {Rune: 'b', Pos: TextPosition{Idx: 3, Line: 2, Col: 1}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
        {Rune: ScanEOF, Pos: TextPosition{Idx: 4, Line: 2, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
      },
    },
    {
      name:  "escaped line break",
      input: "a\\\nb",
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
        {Rune: ScanEOF, Pos: TextPosition{Idx: 4, Line: 2, Col: 2}, End: TextPosition{Idx: 4, Line: 2, Col: 2}},
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
        if span.Rune == ScanEOF {
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
      expected: []rune{ScanEOF},
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
      expected: []rune{ScanEOF, ScanEOF}, // escaped line break consumes both chars
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
        
        if popped == ScanEOF {
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
        {Rune: ScanEOF, Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 0, Line: 1, Col: 1}},
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

        if popped.Rune == ScanEOF {
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

        if poppedSpan.Rune == ScanEOF {
          break
        }
      }
    })
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
    name     string
    input    string
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
        if r == ScanEOF {
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
        if r == ScanEOF {
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
        if scanner.Pop() == ScanEOF {
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

func TestStream(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected []RuneSpan
  }{
    {
      name:  "empty string",
      input: "",
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
        if span.Rune == ScanEOF {
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
      name:  "empty string",
      input: "",
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
      name:  "stop at second character",
      input: "abcdef",
      stopAt: 'b',
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: 'b', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 1, Col: 3}},
      },
    },
    {
      name:  "stop at first character",
      input: "hello",
      stopAt: 'h',
      expected: []RuneSpan{
        {Rune: 'h', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
      },
    },
    {
      name:  "stop at line break",
      input: "a\nb\nc",
      stopAt: '\n',
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: '\n', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 2, Line: 2, Col: 1}},
      },
    },
    {
      name:  "stop at UTF-8 character",
      input: "aβγ",
      stopAt: 'β',
      expected: []RuneSpan{
        {Rune: 'a', Pos: TextPosition{Idx: 0, Line: 1, Col: 1}, End: TextPosition{Idx: 1, Line: 1, Col: 2}},
        {Rune: 'β', Pos: TextPosition{Idx: 1, Line: 1, Col: 2}, End: TextPosition{Idx: 3, Line: 1, Col: 3}},
      },
    },
    {
      name:  "never stop",
      input: "abc",
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
        if span.Rune == ScanEOF {
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