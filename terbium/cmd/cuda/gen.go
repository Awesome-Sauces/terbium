package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var i32 = types.I32
var i8 = types.I8

type MatmulOp struct {
	Out string
	A   string
	B   string
	N   int
}

type MatrixInfo struct {
	Name   string
	N      int
	Alloca value.Value
	Ty     types.Type
}

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

func parseProgram(path string) ([]MatmulOp, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	re := regexp.MustCompile(`^\s*matmul\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\s*x\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*(\d+)x(\d+)\s*$`)

	var ops []MatmulOp

	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		m := re.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("syntax error on line %d: %q", lineNo, line)
		}

		n1, _ := strconv.Atoi(m[4])
		n2, _ := strconv.Atoi(m[5])

		if n1 != n2 {
			return nil, fmt.Errorf("only square matmuls are supported on line %d: %dx%d", lineNo, n1, n2)
		}

		if n1 != 2 && n1 != 4 && n1 != 8 {
			return nil, fmt.Errorf("only 2x2, 4x4, and 8x8 are supported on line %d", lineNo)
		}

		ops = append(ops, MatmulOp{
			Out: m[1],
			A:   m[2],
			B:   m[3],
			N:   n1,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("no matmul operations found")
	}

	return ops, nil
}

func inferMatrices(ops []MatmulOp) (map[string]int, []string, []string, error) {
	sizes := map[string]int{}
	produced := map[string]bool{}
	usedAsInput := map[string]bool{}

	setSize := func(name string, n int) error {
		if old, ok := sizes[name]; ok && old != n {
			return fmt.Errorf("matrix %s used with conflicting sizes: %dx%d and %dx%d", name, old, old, n, n)
		}
		sizes[name] = n
		return nil
	}

	for _, op := range ops {
		if err := setSize(op.Out, op.N); err != nil {
			return nil, nil, nil, err
		}
		if err := setSize(op.A, op.N); err != nil {
			return nil, nil, nil, err
		}
		if err := setSize(op.B, op.N); err != nil {
			return nil, nil, nil, err
		}

		produced[op.Out] = true
		usedAsInput[op.A] = true
		usedAsInput[op.B] = true
	}

	var inputNames []string
	var outputNames []string

	for name := range usedAsInput {
		if !produced[name] {
			inputNames = append(inputNames, name)
		}
	}

	for _, op := range ops {
		outputNames = append(outputNames, op.Out)
	}

	sort.Strings(inputNames)

	return sizes, inputNames, outputNames, nil
}

func generateCUDAFunction(n int) string {
	total := n * n

	var b strings.Builder

	fmt.Fprintf(&b, "__global__ void matmul%dx%d_kernel(const int32_t* A, const int32_t* B, int32_t* C) {\n", n, n)
	b.WriteString(`    int row = threadIdx.y + blockIdx.y * blockDim.y;
    int col = threadIdx.x + blockIdx.x * blockDim.x;

`)
	fmt.Fprintf(&b, "    if (row < %d && col < %d) {\n", n, n)
	b.WriteString("        int32_t sum = 0;\n")
	fmt.Fprintf(&b, "        for (int k = 0; k < %d; k++) {\n", n)
	fmt.Fprintf(&b, "            sum += A[row * %d + k] * B[k * %d + col];\n", n, n)
	b.WriteString("        }\n")
	fmt.Fprintf(&b, "        C[row * %d + col] = sum;\n", n)
	b.WriteString("    }\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(
		&b,
		"extern \"C\" void matmul%dx%d(int32_t (*a)[%d], int32_t (*b)[%d], int32_t (*c)[%d]) {\n",
		n,
		n,
		total,
		total,
		total,
	)

	fmt.Fprintf(&b, "    const size_t bytes = %d * sizeof(int32_t);\n\n", total)

	b.WriteString(`    int32_t* dA = NULL;
    int32_t* dB = NULL;
    int32_t* dC = NULL;

    CUDA_CHECK(cudaMalloc((void**)&dA, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dB, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dC, bytes));

    CUDA_CHECK(cudaMemcpy(dA, *a, bytes, cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaMemcpy(dB, *b, bytes, cudaMemcpyHostToDevice));

`)

	b.WriteString("    dim3 block(16, 16);\n")
	fmt.Fprintf(&b, "    dim3 grid((%d + block.x - 1) / block.x, (%d + block.y - 1) / block.y);\n\n", n, n)

	fmt.Fprintf(&b, "    matmul%dx%d_kernel<<<grid, block>>>(dA, dB, dC);\n\n", n, n)

	b.WriteString(`    CUDA_CHECK(cudaGetLastError());
    CUDA_CHECK(cudaDeviceSynchronize());

    CUDA_CHECK(cudaMemcpy(*c, dC, bytes, cudaMemcpyDeviceToHost));

    CUDA_CHECK(cudaFree(dA));
    CUDA_CHECK(cudaFree(dB));
    CUDA_CHECK(cudaFree(dC));
}

`)

	return b.String()
}

func generateCUDAFile() string {
	var b strings.Builder

	b.WriteString(`#include <cuda_runtime.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#define CUDA_CHECK(call)                                                   \
    do {                                                                   \
        cudaError_t err__ = (call);                                        \
        if (err__ != cudaSuccess) {                                        \
            fprintf(stderr,                                                \
                    "CUDA error at %s:%d: %s\n",                          \
                    __FILE__,                                              \
                    __LINE__,                                              \
                    cudaGetErrorString(err__));                            \
            exit(1);                                                       \
        }                                                                  \
    } while (0)

`)

	// Generate all supported kernels, even if only some are used by the DSL.
	for _, n := range []int{2, 4, 8} {
		b.WriteString(generateCUDAFunction(n))
	}

	return b.String()
}

func generateLLVMHost(ops []MatmulOp) (string, error) {
	matrixSizes, inputNames, outputNames, err := inferMatrices(ops)
	if err != nil {
		return "", err
	}

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

	mainFn := m.NewFunc("main", i32)
	entry := mainFn.NewBlock("entry")

	matrixInfos := map[string]MatrixInfo{}
	matmulDecls := map[int]*ir.Func{}

	// Declare CUDA-backed functions.
	for _, n := range []int{2, 4, 8} {
		total := n * n
		matTy := types.NewArray(uint64(total), i32)
		matPtrTy := types.NewPointer(matTy)

		fnName := fmt.Sprintf("matmul%dx%d", n, n)

		matmulDecls[n] = m.NewFunc(
			fnName,
			types.Void,
			ir.NewParam("a", matPtrTy),
			ir.NewParam("b", matPtrTy),
			ir.NewParam("c", matPtrTy),
		)
	}

	// Allocate every matrix needed by the program.
	var matrixNames []string
	for name := range matrixSizes {
		matrixNames = append(matrixNames, name)
	}
	sort.Strings(matrixNames)

	for _, name := range matrixNames {
		n := matrixSizes[name]
		total := n * n

		matTy := types.NewArray(uint64(total), i32)
		alloca := entry.NewAlloca(matTy)
		alloca.SetName(name)

		matrixInfos[name] = MatrixInfo{
			Name:   name,
			N:      n,
			Alloca: alloca,
			Ty:     matTy,
		}
	}

	// Prompt.
	var promptBuilder strings.Builder
	promptBuilder.WriteString("Input matrices required:\n")

	for _, name := range inputNames {
		n := matrixSizes[name]
		fmt.Fprintf(&promptBuilder, "  %s: enter %d ints for %dx%d\n", name, n*n, n, n)
	}

	prompt := cString(m, entry, "prompt", promptBuilder.String())
	entry.NewCall(printf, prompt)

	// Scan all leaf input matrices.
	var scanSpecs []string
	var scanArgs []value.Value

	for _, name := range inputNames {
		info := matrixInfos[name]
		total := int64(info.N * info.N)

		for i := int64(0); i < total; i++ {
			scanSpecs = append(scanSpecs, "%d")
			scanArgs = append(scanArgs, gepMat(entry, info.Ty, info.Alloca, i))
		}
	}

	scanFmt := cString(m, entry, "scan_fmt", strings.Join(scanSpecs, " "))
	finalScanArgs := []value.Value{scanFmt}
	finalScanArgs = append(finalScanArgs, scanArgs...)

	entry.NewCall(scanf, finalScanArgs...)

	// Emit matmul calls in program order.
	for _, op := range ops {
		aInfo := matrixInfos[op.A]
		bInfo := matrixInfos[op.B]
		cInfo := matrixInfos[op.Out]

		entry.NewCall(
			matmulDecls[op.N],
			aInfo.Alloca,
			bInfo.Alloca,
			cInfo.Alloca,
		)
	}

	// Print outputs.
	for outputIndex, name := range outputNames {
		info := matrixInfos[name]

		headerName := fmt.Sprintf("out_header_%d", outputIndex)
		headerText := fmt.Sprintf("%s =\n", name)
		header := cString(m, entry, headerName, headerText)
		entry.NewCall(printf, header)

		elemFmt := cString(m, entry, fmt.Sprintf("elem_fmt_%d", outputIndex), "%d ")
		newlineFmt := cString(m, entry, fmt.Sprintf("newline_fmt_%d", outputIndex), "\n")

		for row := int64(0); row < int64(info.N); row++ {
			for col := int64(0); col < int64(info.N); col++ {
				idx := row*int64(info.N) + col
				x := loadMat(entry, info.Ty, info.Alloca, idx)
				entry.NewCall(printf, elemFmt, x)
			}
			entry.NewCall(printf, newlineFmt)
		}
	}

	entry.NewRet(ci(0))

	return m.String(), nil
}

func main() {
	ops, err := parseProgram("program.txt")
	if err != nil {
		panic(err)
	}

	llvmHost, err := generateLLVMHost(ops)
	if err != nil {
		panic(err)
	}

	cudaSource := generateCUDAFile()

	if err := os.WriteFile("main.ll", []byte(llvmHost), 0644); err != nil {
		panic(err)
	}

	if err := os.WriteFile("matmul_generated.cu", []byte(cudaSource), 0644); err != nil {
		panic(err)
	}

	fmt.Println("Generated:")
	fmt.Println("  main.ll")
	fmt.Println("  matmul_generated.cu")
}
