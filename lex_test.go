package lex

import (
	"errors"
	"testing"
)

const (
	TokWord TokenType = iota
)

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

func lexWord(lex *Lexer) StateFn {
	for {
		r := lex.Peek()
		if isSpace(r) {
			lex.Emit(TokWord)
			return lexSkipSpaces
		} else if r == EOF {
			lex.Emit(TokWord)
			return nil
		}
		lex.Next()
	}
}

func lexSkipSpaces(lex *Lexer) StateFn {
	for isSpace(lex.Next()) {
	}
	lex.Backup()
	lex.Ignore()
	return lexWord
}

func TestWordReader(t *testing.T) {
	lex := NewLexerString("  foo bar\n    baz    ")
	c := lex.Run(lexSkipSpaces)

	foo := <-c
	if foo.Value != "foo" || foo.Line != 0 || foo.Col != 2 || foo.Pos != 2 {
		t.Error(foo)
	}

	bar := <-c
	if bar.Value != "bar" || bar.Line != 0 || bar.Col != 6 || bar.Pos != 6 {
		t.Error(bar)
	}

	baz := <-c
	if baz.Value != "baz" || baz.Line != 1 || baz.Col != 4 || baz.Pos != 14 {
		t.Error(baz)
	}

	if tok, ok := <-c; ok {
		t.Error("token channel should be closed, but got", tok)
	}
}

type brokenReader struct{ data string }

func (r *brokenReader) Read(p []byte) (int, error) {
	if r.data == "" {
		return 0, errors.New("broken")
	} else {
		b := []byte(r.data)
		copy(p, b)
		r.data = ""
		return len([]byte(b)), nil
	}
}

func TestBrokenReader(t *testing.T) {
	// foo will be read, bar will return an error
	lex := NewLexer(&brokenReader{"foo bar"})
	c := lex.Run(lexSkipSpaces)

	foo := <-c
	if foo.Value != "foo" || foo.Line != 0 || foo.Col != 0 || foo.Pos != 0 {
		t.Error(foo)
	}

	err := <-c
	if err.Type != TokError {
		t.Error(err)
	}
	if err.Value != "broken" || err.Line != 0 || err.Col != 7 || err.Pos != 7 {
		t.Error(err)
	}
	for range c {
	}
}

func TestTokenString(t *testing.T) {
	lex := NewLexerString("\n \nfoo")
	c := lex.Run(lexSkipSpaces)
	foo := <-c
	if foo.Value != "foo" || foo.Line != 2 || foo.Col != 0 || foo.Pos != 3 {
		t.Error(foo)
	}
	if foo.String() != `2:0(3): token=0 "foo"` {
		t.Error(foo.String())
	}
	for range c {
	}
}
