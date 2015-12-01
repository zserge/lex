package lex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type TokenType int

const EOF rune = -1
const TokError TokenType = -1

type pos struct {
	line int
	col  int
	pos  int
}

func (p *pos) CopyTo(to *pos) {
	to.line = p.line
	to.col = p.col
	to.pos = p.pos
}

func (p *pos) Advance(r rune) {
	p.col++
	p.pos++
	if r == '\n' {
		p.line++
		p.col = 0
	}
}

type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
	Pos   int
	Extra interface{}
}

func (tok *Token) String() string {
	return fmt.Sprintf("%d:%d(%d): token=%d %q",
		tok.Line, tok.Col, tok.Pos, tok.Type, tok.Value)
}

type StateFn func(Lexer) StateFn

type Lexer interface {
	Backup()
	Col() int
	Emit(t TokenType)
	EmitExtra(t TokenType, extra interface{})
	Errorf(t TokenType, s string, args ...interface{}) StateFn
	Ignore()
	Line() int
	Next() rune
	Peek() rune
	Pos() int
	Run(start StateFn) <-chan Token
	Value() string
}

type lexer struct {
	r      *bufio.Reader
	tokens chan Token
	eof    bool
	// currently bufferred value
	value []rune
	// Position in the stream
	pos      pos
	prevPos  pos
	tokenPos pos
}

func NewLexer(r io.Reader) Lexer {
	return &lexer{
		r:      bufio.NewReader(r),
		tokens: make(chan Token, 0),
	}
}

func NewLexerString(s string) Lexer {
	return NewLexer(bytes.NewBufferString(s))
}

func (lex *lexer) Next() rune {
	if r, _, err := lex.r.ReadRune(); err != nil {
		if err != io.EOF {
			lex.Errorf(TokError, err.Error())
		}
		lex.value = append(lex.value, r)
		lex.eof = true
		return EOF
	} else {
		lex.pos.CopyTo(&lex.prevPos)
		lex.pos.Advance(r)
		lex.value = append(lex.value, r)
		return r
	}
}

func (lex *lexer) Peek() rune {
	r := lex.Next()
	lex.Backup()
	return r
}

func (lex *lexer) Backup() {
	lex.r.UnreadRune()
	lex.value = lex.value[0 : len(lex.value)-1]
	lex.prevPos.CopyTo(&lex.pos)
}

// Line() returns current line number in the reader
func (lex *lexer) Line() int {
	return lex.pos.line
}

// Line() returns current column number in the current line of reader
func (lex *lexer) Col() int {
	return lex.pos.col
}

// Line() returns current position in the reader (in runes)
func (lex *lexer) Pos() int {
	return lex.pos.pos
}

// Value() returns currently buffered token value
func (lex *lexer) Value() string {
	return string(lex.value)
}

// Ignore() removes currently buffered token value
func (lex *lexer) Ignore() {
	lex.pos.CopyTo(&lex.tokenPos)
	lex.value = []rune{}
}

func (lex *lexer) Emit(t TokenType) {
	lex.EmitExtra(t, nil)
}

func (lex *lexer) EmitExtra(t TokenType, extra interface{}) {
	lex.tokens <- Token{t, lex.Value(), lex.tokenPos.line, lex.tokenPos.col, lex.tokenPos.pos, extra}
	lex.pos.CopyTo(&lex.tokenPos)
	lex.value = []rune{}
}

func (lex *lexer) Errorf(t TokenType, s string, args ...interface{}) StateFn {
	value := fmt.Sprintf(s, args...)
	lex.tokens <- Token{t, value, lex.Line(), lex.Col(), lex.Pos(), nil}
	return nil
}

func (lex *lexer) Run(start StateFn) <-chan Token {
	go func() {
		for state := start; state != nil && !lex.eof; {
			state = state(lex)
		}
		close(lex.tokens)
	}()
	return lex.tokens
}
