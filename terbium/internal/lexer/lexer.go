package lexer

import (
	"fmt"
	"os"
	"unicode"

	"github.com/Awesome-Sauces/terbium/internal/log"
)

type Token struct {
	Type     uint8
	Literal  []byte
	Children []Token
}

const (
	IDENTIFIER = iota
	PARENTHETICAL_CONTAINER
	SQUARE_BRACKET_CONTAINER
	CURLY_BRACE_CONTAINER
	STRING_CONTAINER
	NUMBER
	STRING
	KEYWORD
	OPERATOR
	EOF
	NIL
)

func TokenTypeToString(Type uint8) string {
	switch Type {
	case IDENTIFIER:
		return "IDENTIFIER"
	case PARENTHETICAL_CONTAINER:
		return "PARENTHETICAL_CONTAINER"
	case SQUARE_BRACKET_CONTAINER:
		return "SQUARE_BRACKET_CONTAINER"
	case CURLY_BRACE_CONTAINER:
		return "CURLY_BRACE_CONTAINER"
	case STRING_CONTAINER:
		return "STRING_CONTAINER"
	case NUMBER:
		return "NUMBER"
	case STRING:
		return "STRING"
	case KEYWORD:
		return "KEYWORD"
	case OPERATOR:
		return "OPERATOR"
	case EOF:
		return "EOF"
	case NIL:
		return "NIL"
	default:
		return "UNKNOWN"
	}
}

type lexer struct {
	src  []byte
	pos  int
	line int
	col  int
}

func Lex(filepath string) ([]Token, error) {
	src, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	log.Trace(fmt.Sprintf("Scanning file %s", filepath))

	l := &lexer{
		src:  src,
		pos:  0,
		line: 1,
		col:  1,
	}

	tokens, err := l.lexUntil(0)
	if err != nil {
		return nil, err
	}

	tokens = append(tokens, Token{
		Type:    EOF,
		Literal: []byte{},
	})

	return tokens, nil
}

func (l *lexer) lexUntil(closing byte) ([]Token, error) {
	tokens := []Token{}

	for !l.atEnd() {
		l.skipWhitespaceAndComments()

		if l.atEnd() {
			break
		}

		c := l.peek()

		if closing != 0 && c == closing {
			l.advance()
			return tokens, nil
		}

		switch c {
		case '(':
			tok, err := l.lexContainer('(', ')', PARENTHETICAL_CONTAINER)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case '[':
			tok, err := l.lexContainer('[', ']', SQUARE_BRACKET_CONTAINER)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case '{':
			tok, err := l.lexContainer('{', '}', CURLY_BRACE_CONTAINER)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case ')', ']', '}':
			return nil, l.errf("unexpected closing delimiter %q", c)

		case '"':
			tok, err := l.lexQuoted('"', STRING_CONTAINER)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case '\'':
			tok, err := l.lexQuoted('\'', STRING)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case '#':
			tokens = append(tokens, l.lexDirectiveOrOperator())

		default:
			if isIdentStart(c) {
				tokens = append(tokens, l.lexIdentifierOrKeyword())
			} else if isDigit(c) {
				tokens = append(tokens, l.lexNumber())
			} else {
				tokens = append(tokens, l.lexOperator())
			}
		}
	}

	if closing != 0 {
		return nil, l.errf("expected closing delimiter %q before EOF", closing)
	}

	return tokens, nil
}

func (l *lexer) lexContainer(opening byte, closing byte, tokenType uint8) (Token, error) {
	l.advance()

	children, err := l.lexUntil(closing)
	if err != nil {
		return Token{}, err
	}

	return Token{
		Type:     tokenType,
		Literal:  []byte{opening, closing},
		Children: children,
	}, nil
}

func (l *lexer) lexQuoted(quote byte, tokenType uint8) (Token, error) {
	l.advance()

	literal := []byte{}

	for !l.atEnd() {
		c := l.peek()

		if c == quote {
			l.advance()
			return Token{
				Type:    tokenType,
				Literal: literal,
			}, nil
		}

		if c == '\\' {
			l.advance()

			if l.atEnd() {
				return Token{}, l.errf("unterminated escape sequence")
			}

			escaped := l.peek()
			literal = append(literal, '\\', escaped)
			l.advance()
			continue
		}

		literal = append(literal, c)
		l.advance()
	}

	return Token{}, l.errf("unterminated string literal")
}

func (l *lexer) lexDirectiveOrOperator() Token {
	start := l.pos

	l.advance()

	for !l.atEnd() && isIdentPart(l.peek()) {
		l.advance()
	}

	lit := l.src[start:l.pos]

	if isKeyword(lit) {
		return Token{
			Type:    KEYWORD,
			Literal: lit,
		}
	}

	return Token{
		Type:    OPERATOR,
		Literal: lit,
	}
}

func (l *lexer) lexIdentifierOrKeyword() Token {
	start := l.pos

	for !l.atEnd() && isIdentPart(l.peek()) {
		l.advance()
	}

	lit := l.src[start:l.pos]

	if isKeyword(lit) {
		return Token{
			Type:    KEYWORD,
			Literal: lit,
		}
	}

	return Token{
		Type:    IDENTIFIER,
		Literal: lit,
	}
}

func (l *lexer) lexNumber() Token {
	start := l.pos

	for !l.atEnd() {
		c := l.peek()

		if isDigit(c) || c == '_' {
			l.advance()
			continue
		}

		if c == '.' && l.peekNextIsDigit() {
			l.advance()
			continue
		}

		break
	}

	return Token{
		Type:    NUMBER,
		Literal: l.src[start:l.pos],
	}
}

func (l *lexer) lexOperator() Token {
	operators := []string{
		"::",
		"==",
		"!=",
		"<=",
		">=",
		"&&",
		"||",
		"+=",
		"-=",
		"*=",
		"/=",
		"%=",
		"++",
		"--",
		"->",
		"=>",
	}

	for _, op := range operators {
		if l.matchString(op) {
			start := l.pos
			for range op {
				l.advance()
			}

			return Token{
				Type:    OPERATOR,
				Literal: l.src[start:l.pos],
			}
		}
	}

	c := l.peek()
	l.advance()

	return Token{
		Type:    OPERATOR,
		Literal: []byte{c},
	}
}

func (l *lexer) skipWhitespaceAndComments() {
	for !l.atEnd() {
		c := l.peek()

		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			l.advance()
			continue
		}

		if l.matchString("//") {
			for !l.atEnd() && l.peek() != '\n' {
				l.advance()
			}
			continue
		}

		if l.matchString("/*") {
			l.advance()
			l.advance()

			for !l.atEnd() && !l.matchString("*/") {
				l.advance()
			}

			if !l.atEnd() {
				l.advance()
				l.advance()
			}

			continue
		}

		break
	}
}

func (l *lexer) advance() byte {
	c := l.src[l.pos]
	l.pos++

	if c == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}

	return c
}

func (l *lexer) peek() byte {
	return l.src[l.pos]
}

func (l *lexer) atEnd() bool {
	return l.pos >= len(l.src)
}

func (l *lexer) matchString(s string) bool {
	if l.pos+len(s) > len(l.src) {
		return false
	}

	for i := 0; i < len(s); i++ {
		if l.src[l.pos+i] != s[i] {
			return false
		}
	}

	return true
}

func (l *lexer) peekNextIsDigit() bool {
	if l.pos+1 >= len(l.src) {
		return false
	}

	return isDigit(l.src[l.pos+1])
}

func (l *lexer) errf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("lexer error at %d:%d: %s", l.line, l.col, msg)
}

func isIdentStart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c))
}

func isIdentPart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isKeyword(lit []byte) bool {
	switch string(lit) {
	case
		"#import",
		"main",
		"void",
		"int",
		"float",
		"double",
		"bool",
		"char",
		"String",
		"object",
		"extends",
		"private",
		"public",
		"new",
		"for",
		"in",
		"if",
		"else",
		"return",
		"continue",
		"break",
		"true",
		"false",
		"nil",
		"null",
		"exit":
		return true
	default:
		return false
	}
}
