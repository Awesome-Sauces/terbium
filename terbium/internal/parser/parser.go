package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Awesome-Sauces/terbium/internal/ast"
)

func Parse(src string) (*ast.Program, error) {
	fields := strings.Fields(src)

	if len(fields) < 2 || fields[0] != "print" {
		return nil, fmt.Errorf("expected: print <expression>")
	}

	expr, err := parseExpr(fields[1:])
	if err != nil {
		return nil, err
	}

	return &ast.Program{
		Statements: []ast.Stmt{
			&ast.PrintStmt{
				Value: expr,
			},
		},
	}, nil
}

func parseExpr(fields []string) (ast.Expr, error) {
	if len(fields) == 1 {
		return parseInt(fields[0])
	}

	if len(fields) == 3 {
		left, err := parseInt(fields[0])
		if err != nil {
			return nil, err
		}

		op := fields[1]

		right, err := parseInt(fields[2])
		if err != nil {
			return nil, err
		}

		switch op {
			case "+", "-", "*", "/":
				return &ast.BinaryExpr{
					Op:    op,
					Left:  left,
					Right: right,
				}, nil
			default:
				return nil, fmt.Errorf("unknown operator %q", op)
		}
	}

	return nil, fmt.Errorf("unsupported expression: %q", strings.Join(fields, " "))
}

func parseInt(s string) (*ast.IntLiteral, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer %q", s)
	}

	return &ast.IntLiteral{Value: n}, nil
}
