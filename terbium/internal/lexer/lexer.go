package lexer

import (
	"bufio"
	"fmt"
	"strings"
	"unicode"

	"github.com/Awesome-Sauces/terbium/internal/log"
)

type Token struct {
	Type    TokenType
	Literal string
}

type TokenType string

const (
	ILLEGAL       TokenType = "ILLEGAL"
	EOF           TokenType = "EOF"
	PREPDIRECTIVE TokenType = "PREPROCESSOR_DIRECTIVE"
	METHOD        TokenType = "METHOD" // another word for a function
	PRIVATE       TokenType = "PRIVATE"
	PUBLIC        TokenType = "PUBLIC"
	STRINGLITERAL TokenType = "STRING_LITERAL"
	INTLITERAL    TokenType = "INT_LITERAL"
	FLOATLITERAL  TokenType = "FLOAT_LITERAL"
	IDENTIFIER    TokenType = "IDENTIFIER"
	OPERATOR      TokenType = "OPERATOR"
	DELIMITER     TokenType = "DELIMITER"
	LCBRACKET     TokenType = "LEFT_CURLY_BRACKET"
	RCBRACKET     TokenType = "RIGHT_CURLY_BRACKET"
	LPAREN        TokenType = "LEFT_PAREN"
	RPAREN        TokenType = "RIGHT_PAREN"
	SEMICOLON     TokenType = "SEMICOLON" // when lexing we need to treat this as the start of a "newline"
	LSBRACKET     TokenType = "LEFT_SQUARE_BRACKET"
	RSBRACKET     TokenType = "RIGHT_SQUARE_BRACKET"
)

var keywords = map[string]TokenType{
	"method":  METHOD,
	"private": PRIVATE,
	"public":  PUBLIC,
}

var multiCharOperators = map[string]struct{}{
	"==": {},
	"!=": {},
	"<=": {},
	">=": {},
	"&&": {},
	"||": {},
	"+=": {},
	"-=": {},
	"*=": {},
	"/=": {},
	"%=": {},
	"++": {},
	"--": {},
	"->": {},
	"=>": {},
	"::": {},
	"**": {},
}

func isBlankLine(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" || strings.HasPrefix(line, "//")
}

func isComment(field string) bool {
	return strings.HasPrefix(strings.TrimSpace(field), "//")
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

func isOperatorChar(ch rune) bool {
	switch ch {
	case '+', '-', '*', '/', '%', '=', '!', '<', '>', '&', '|', '^', '~':
		return true
	default:
		return false
	}
}

func isDelimiterChar(ch rune) bool {
	switch ch {
	case ',', ':', '.':
		return true
	default:
		return false
	}
}

func tokenForSingleChar(ch rune) TokenType {
	switch ch {
	case '{':
		return LCBRACKET
	case '}':
		return RCBRACKET
	case '(':
		return LPAREN
	case ')':
		return RPAREN
	case '[':
		return LSBRACKET
	case ']':
		return RSBRACKET
	case ';':
		return SEMICOLON
	default:
		return ILLEGAL
	}
}

func emitToken(tokens *[]Token, lineNumber int, token Token) {
	*tokens = append(*tokens, token)

	log.Info(
		fmt.Sprintf("L%d token", lineNumber),
		"type", token.Type,
		"literal", token.Literal,
	)
}

func Lex(source string) ([]Token, error) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Split(bufio.ScanLines)

	tokens := []Token{}
	lineNumber := 1

	for scanner.Scan() {
		line := scanner.Text()

		log.Info(
			fmt.Sprintf("L%d", lineNumber),
			"literal", line,
		)

		if isBlankLine(line) {
			log.Info(
				fmt.Sprintf("L%d skipped", lineNumber),
				"reason", "blank line or full-line comment",
			)

			lineNumber++
			continue
		}

		runes := []rune(line)
		i := 0

		for i < len(runes) {
			ch := runes[i]

			if unicode.IsSpace(ch) {
				i++
				continue
			}

			// Line comment
			if ch == '/' && i+1 < len(runes) && runes[i+1] == '/' {
				log.Info(
					fmt.Sprintf("L%d comment", lineNumber),
					"literal", string(runes[i:]),
				)
				break
			}

			// Preprocessor directive.
			// Consumes the rest of the line.
			if ch == '#' {
				literal := strings.TrimSpace(string(runes[i:]))

				emitToken(&tokens, lineNumber, Token{
					Type:    PREPDIRECTIVE,
					Literal: literal,
				})

				break
			}

			// String literal
			if ch == '"' {
				start := i
				i++ // consume opening quote

				escaped := false
				closed := false

				for i < len(runes) {
					if escaped {
						escaped = false
						i++
						continue
					}

					if runes[i] == '\\' {
						escaped = true
						i++
						continue
					}

					if runes[i] == '"' {
						i++ // consume closing quote
						closed = true
						break
					}

					i++
				}

				literal := string(runes[start:i])
				tokenType := STRINGLITERAL

				if !closed {
					tokenType = ILLEGAL

					log.Info(
						fmt.Sprintf("L%d lexer error", lineNumber),
						"reason", "unterminated string literal",
						"literal", literal,
					)
				}

				emitToken(&tokens, lineNumber, Token{
					Type:    tokenType,
					Literal: literal,
				})

				continue
			}

			// Number literal: int or float
			if unicode.IsDigit(ch) {
				start := i
				hasDot := false

				for i < len(runes) {
					if unicode.IsDigit(runes[i]) {
						i++
						continue
					}

					if runes[i] == '.' && !hasDot {
						if i+1 < len(runes) && runes[i+1] == '.' {
							break
						}

						hasDot = true
						i++
						continue
					}

					break
				}

				tokenType := INTLITERAL
				if hasDot {
					tokenType = FLOATLITERAL
				}

				emitToken(&tokens, lineNumber, Token{
					Type:    tokenType,
					Literal: string(runes[start:i]),
				})

				continue
			}

			// Identifier or keyword
			if isIdentifierStart(ch) {
				start := i
				i++

				for i < len(runes) && isIdentifierPart(runes[i]) {
					i++
				}

				literal := string(runes[start:i])
				tokenType, ok := keywords[literal]
				if !ok {
					tokenType = IDENTIFIER
				}

				emitToken(&tokens, lineNumber, Token{
					Type:    tokenType,
					Literal: literal,
				})

				continue
			}

			// Brackets, parentheses, semicolon
			if tokenType := tokenForSingleChar(ch); tokenType != ILLEGAL {
				emitToken(&tokens, lineNumber, Token{
					Type:    tokenType,
					Literal: string(ch),
				})

				i++
				continue
			}

			// Operators, including multi-character operators
			if isOperatorChar(ch) {
				if i+1 < len(runes) {
					two := string(runes[i : i+2])

					if _, ok := multiCharOperators[two]; ok {
						emitToken(&tokens, lineNumber, Token{
							Type:    OPERATOR,
							Literal: two,
						})

						i += 2
						continue
					}
				}

				emitToken(&tokens, lineNumber, Token{
					Type:    OPERATOR,
					Literal: string(ch),
				})

				i++
				continue
			}

			// Delimiters
			if isDelimiterChar(ch) {
				emitToken(&tokens, lineNumber, Token{
					Type:    DELIMITER,
					Literal: string(ch),
				})

				i++
				continue
			}

			// Anything unknown
			emitToken(&tokens, lineNumber, Token{
				Type:    ILLEGAL,
				Literal: string(ch),
			})

			log.Info(
				fmt.Sprintf("L%d lexer warning", lineNumber),
				"reason", "unknown character",
				"literal", string(ch),
			)

			i++
		}

		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	emitToken(&tokens, lineNumber, Token{
		Type:    EOF,
		Literal: "",
	})

	log.Info(
		"lexer complete",
		"tokens", len(tokens),
	)

	return tokens, nil
}
