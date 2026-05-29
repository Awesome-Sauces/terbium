package lexer

type WhitespaceKind uint8

const (
	WSNone WhitespaceKind = iota
	WSSpace
	WSTab
	WSLF
	WSCRLF
)

// ClassifyWhitespace only recognizes:
//   - space: ' '
//   - tab: '\t'
//   - LF: '\n'
//   - CRLF: "\r\n"
//
// A bare '\r' is not treated as whitespace.
func ClassifyWhitespace(src []byte, i int) (kind WhitespaceKind, width int) {
	if i >= len(src) {
		return WSNone, 0
	}

	switch src[i] {
	case ' ':
		return WSSpace, 1
	case '\t':
		return WSTab, 1
	case '\n':
		return WSLF, 1
	case '\r':
		if i+1 < len(src) && src[i+1] == '\n' {
			return WSCRLF, 2
		}
		return WSNone, 0
	default:
		return WSNone, 0
	}
}

func IsHorizontalWhitespace(kind WhitespaceKind) bool {
	return kind == WSSpace || kind == WSTab
}

func IsLineTerminator(kind WhitespaceKind) bool {
	return kind == WSLF || kind == WSCRLF
}
