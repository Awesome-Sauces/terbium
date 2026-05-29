package main

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var i32 = types.I32
var i8 = types.I8

func ci(x int64) *constant.Int {
	return constant.NewInt(i32, x)
}

func cString(m *ir.Module, block *ir.Block, name string, text string) value.Value {
	strConst := constant.NewCharArrayFromString(text + "\x00")
	g := m.NewGlobalDef(name, strConst)

	return block.NewGetElementPtr(
		strConst.Typ,
		g,
		ci(0),
		ci(0),
	)
}

func gepMat(block *ir.Block, matTy types.Type, matrix value.Value, index int64) value.Value {
	return block.NewGetElementPtr(matTy, matrix, ci(0), ci(index))
}

func loadMat(block *ir.Block, matTy types.Type, matrix value.Value, index int64) value.Value {
	ptr := gepMat(block, matTy, matrix, index)
	return block.NewLoad(i32, ptr)
}

func main() {
	m := ir.NewModule()

	printf := m.NewFunc(
		"printf",
		i32,
		ir.NewParam("format", types.NewPointer(i8)),
	)
	printf.Sig.Variadic = true

	scanf := m.NewFunc(
		"scanf",
		i32,
		ir.NewParam("format", types.NewPointer(i8)),
	)
	scanf.Sig.Variadic = true

	matTy := types.NewArray(4, i32)
	matPtrTy := types.NewPointer(matTy)

	// This declaration will be resolved by the CUDA object file.
	//
	// C ABI equivalent:
	// extern "C" void matmul2x2(int32_t (*a)[4], int32_t (*b)[4], int32_t (*c)[4]);
	matmul := m.NewFunc(
		"matmul2x2",
		types.Void,
		ir.NewParam("a", matPtrTy),
		ir.NewParam("b", matPtrTy),
		ir.NewParam("c", matPtrTy),
	)

	mainFn := m.NewFunc("main", i32)
	entry := mainFn.NewBlock("entry")

	A := entry.NewAlloca(matTy)
	A.SetName("A")

	B := entry.NewAlloca(matTy)
	B.SetName("B")

	C := entry.NewAlloca(matTy)
	C.SetName("C")

	prompt := cString(
		m,
		entry,
		"prompt",
		"Enter 8 ints: A00 A01 A10 A11 B00 B01 B10 B11\n",
	)
	entry.NewCall(printf, prompt)

	scanFmt := cString(
		m,
		entry,
		"scan_fmt",
		"%d %d %d %d %d %d %d %d",
	)

	a00 := gepMat(entry, matTy, A, 0)
	a01 := gepMat(entry, matTy, A, 1)
	a10 := gepMat(entry, matTy, A, 2)
	a11 := gepMat(entry, matTy, A, 3)

	b00 := gepMat(entry, matTy, B, 0)
	b01 := gepMat(entry, matTy, B, 1)
	b10 := gepMat(entry, matTy, B, 2)
	b11 := gepMat(entry, matTy, B, 3)

	entry.NewCall(
		scanf,
		scanFmt,
		a00, a01, a10, a11,
		b00, b01, b10, b11,
	)

	// Calls the CUDA-backed implementation.
	entry.NewCall(matmul, A, B, C)

	c00 := loadMat(entry, matTy, C, 0)
	c01 := loadMat(entry, matTy, C, 1)
	c10 := loadMat(entry, matTy, C, 2)
	c11 := loadMat(entry, matTy, C, 3)

	printFmt := cString(
		m,
		entry,
		"print_fmt",
		"Result:\n%d %d\n%d %d\n",
	)

	entry.NewCall(printf, printFmt, c00, c01, c10, c11)

	entry.NewRet(ci(0))

	fmt.Println(m)
}
