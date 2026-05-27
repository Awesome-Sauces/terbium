package main

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var i32 = types.I32

func ci(x int64) *constant.Int {
	return constant.NewInt(i32, x)
}

func matIndex(row, col int64) int64 {
	return row*2 + col
}

func gepMat(block *ir.Block, matTy types.Type, matrix value.Value, index int64) value.Value {
	return block.NewGetElementPtr(matTy, matrix, ci(0), ci(index))
}

func loadMat(block *ir.Block, matTy types.Type, matrix value.Value, row, col int64) value.Value {
	ptr := gepMat(block, matTy, matrix, matIndex(row, col))
	return block.NewLoad(i32, ptr)
}

func storeMat(block *ir.Block, matTy types.Type, matrix value.Value, row, col int64, x value.Value) {
	ptr := gepMat(block, matTy, matrix, matIndex(row, col))
	block.NewStore(x, ptr)
}

func main() {
	m := ir.NewModule()

	matTy := types.NewArray(4, i32)
	matPtrTy := types.NewPointer(matTy)

	a := ir.NewParam("a", matPtrTy)
	b := ir.NewParam("b", matPtrTy)
	c := ir.NewParam("c", matPtrTy)

	fn := m.NewFunc("matmul2x2", types.Void, a, b, c)
	entry := fn.NewBlock("entry")

	for row := int64(0); row < 2; row++ {
		for col := int64(0); col < 2; col++ {
			a0 := loadMat(entry, matTy, a, row, 0)
			b0 := loadMat(entry, matTy, b, 0, col)
			p0 := entry.NewMul(a0, b0)

			a1 := loadMat(entry, matTy, a, row, 1)
			b1 := loadMat(entry, matTy, b, 1, col)
			p1 := entry.NewMul(a1, b1)

			sum := entry.NewAdd(p0, p1)

			storeMat(entry, matTy, c, row, col, sum)
		}
	}

	entry.NewRet(nil)

	fmt.Println(m)
}
