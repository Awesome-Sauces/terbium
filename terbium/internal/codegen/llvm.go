package codegen

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"

	"github.com/Awesome-Sauces/terbium/internal/ast"
)

type Generator struct {
	mod     *ir.Module
	main    *ir.Func
	entry   *ir.Block
	printf  *ir.Func
	fmtStr  *ir.Global
	symbols map[string]value.Value
}

func Generate(program *ast.Program) (string, error) {
	g := NewGenerator()

	if err := g.generateProgram(program); err != nil {
		return "", err
	}

	return g.mod.String(), nil
}

func NewGenerator() *Generator {
	mod := ir.NewModule()

	g := &Generator{
		mod:     mod,
		symbols: make(map[string]value.Value),
	}

	g.declarePrintf()
	g.defineFormatString()
	g.defineMain()

	return g
}

func (g *Generator) declarePrintf() {
	// Modern LLVM uses opaque pointers, so ptr is enough.
	// printf: i32 (ptr, ...)
	g.printf = g.mod.NewFunc(
		"printf",
		types.I32,
		ir.NewParam("format", types.NewPointer(types.I8)),
	)

	g.printf.Sig.Variadic = true
}

func (g *Generator) defineFormatString() {
	// "%d\n\0" = 4 bytes.
	str := constant.NewCharArrayFromString("%d\n\x00")

	g.fmtStr = g.mod.NewGlobalDef("fmt", str)
	g.fmtStr.Immutable = true
	g.fmtStr.Linkage = enum.LinkagePrivate
	g.fmtStr.UnnamedAddr = enum.UnnamedAddrUnnamedAddr
}

func (g *Generator) defineMain() {
	g.main = g.mod.NewFunc("main", types.I32)
	g.entry = g.main.NewBlock("entry")
}

func (g *Generator) generateProgram(program *ast.Program) error {
	for _, stmt := range program.Statements {
		if err := g.generateStmt(stmt); err != nil {
			return err
		}
	}

	g.entry.NewRet(constant.NewInt(types.I32, 0))
	return nil
}

func (g *Generator) generateStmt(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.PrintStmt:
		return g.generatePrintStmt(s)
	default:
		return fmt.Errorf("unsupported statement %T", stmt)
	}
}

func (g *Generator) generatePrintStmt(stmt *ast.PrintStmt) error {
	val, err := g.generateExpr(stmt.Value)
	if err != nil {
		return err
	}

	zero := constant.NewInt(types.I64, 0)

	fmtPtr := g.entry.NewGetElementPtr(
		g.fmtStr.ContentType,
		g.fmtStr,
		zero,
		zero,
	)

	g.entry.NewCall(g.printf, fmtPtr, val)
	return nil
}

func (g *Generator) generateExpr(expr ast.Expr) (value.Value, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return constant.NewInt(types.I32, e.Value), nil

	case *ast.BinaryExpr:
		left, err := g.generateExpr(e.Left)
		if err != nil {
			return nil, err
		}

		right, err := g.generateExpr(e.Right)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case "+":
			return g.entry.NewAdd(left, right), nil
		case "-":
			return g.entry.NewSub(left, right), nil
		case "*":
			return g.entry.NewMul(left, right), nil
		case "/":
			return g.entry.NewSDiv(left, right), nil
		default:
			return nil, fmt.Errorf("unknown binary operator %q", e.Op)
		}

	default:
		return nil, fmt.Errorf("unsupported expression %T", expr)
	}
}
