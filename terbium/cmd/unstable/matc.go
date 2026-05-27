package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

var i32 = types.I32

type MatrixLit struct {
	Name string
	Rows [][]int64
}

type ImportStmt struct {
	Name string
	From string
}

type MatmulStmt struct {
	Left  string
	Right string
	Out   string
	Final bool
}

type FileAST struct {
	Path     string
	Imports  []ImportStmt
	Matrices []MatrixLit
	Ops      []MatmulStmt
}

type MatrixInfo struct {
	Name       string
	SymbolBase string
	Rows       int
	Cols       int
	Values     []int64
}

type ModuleResult struct {
	AST     *FileAST
	LLPath  string
	Symbols map[string]MatrixInfo
}

type Compiler struct {
	EntryPath string
	OutName   string
	BuildDir  string
	Modules   map[string]*ModuleResult
	Order     []string
}

func main() {
	if len(os.Args) < 3 || os.Args[1] != "build" {
		fmt.Println("usage:")
		fmt.Println("  go run matc.go build main.matmul -o matprog")
		os.Exit(1)
	}

	entry := os.Args[2]
	out := "a.out"

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "-o" && i+1 < len(os.Args) {
			out = os.Args[i+1]
			i++
		}
	}

	c := &Compiler{
		EntryPath: entry,
		OutName:   out,
		BuildDir:  "matbuild",
		Modules:   map[string]*ModuleResult{},
	}

	if err := c.Build(); err != nil {
		fmt.Fprintf(os.Stderr, "matc error: %v\n", err)
		os.Exit(1)
	}
}

func (c *Compiler) Build() error {
	if err := os.MkdirAll(c.BuildDir, 0755); err != nil {
		return err
	}

	entryAbs, err := filepath.Abs(c.EntryPath)
	if err != nil {
		return err
	}

	if _, err := c.compileFile(entryAbs, true); err != nil {
		return err
	}

	var llFiles []string
	for _, p := range c.Order {
		llFiles = append(llFiles, c.Modules[p].LLPath)
	}

	combined := filepath.Join(c.BuildDir, "combined.ll")
	args := append(llFiles, "-S", "-o", combined)

	if err := run("llvm-link", args...); err != nil {
		return fmt.Errorf("llvm-link failed: %w", err)
	}

	if err := run("clang", combined, "-o", c.OutName); err != nil {
		return fmt.Errorf("clang failed: %w", err)
	}

	fmt.Printf("built %s\n", c.OutName)
	fmt.Printf("linked IR: %s\n", combined)
	return nil
}

func run(name string, args ...string) error {
	fmt.Printf("+ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ---------- parser ----------

func parseFile(path string) (*FileAST, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ast := &FileAST{Path: path}
	sc := bufio.NewScanner(f)

	importRe := regexp.MustCompile(`^import\s+"([^"]+)"\s+from\s+"?([^"]+)"?$`)
	matrixStartRe := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\[$`)
	matrixStartSameLineRe := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\[\s*(.*)$`)
	assignMatmulRe := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*@\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*([A-Za-z_][A-Za-z0-9_]*)$`)
	finalMatmulRe := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*@\s*([A-Za-z_][A-Za-z0-9_]*)$`)

	lineNo := 0

	for sc.Scan() {
		lineNo++
		line := clean(sc.Text())
		if line == "" {
			continue
		}

		if m := importRe.FindStringSubmatch(line); m != nil {
			ast.Imports = append(ast.Imports, ImportStmt{
				Name: m[1],
				From: m[2],
			})
			continue
		}

		if m := matrixStartRe.FindStringSubmatch(line); m != nil {
			rows, err := readMatrixRows(sc, &lineNo, "")
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
			}
			ast.Matrices = append(ast.Matrices, MatrixLit{Name: m[1], Rows: rows})
			continue
		}

		if m := matrixStartSameLineRe.FindStringSubmatch(line); m != nil {
			rows, err := readMatrixRows(sc, &lineNo, strings.TrimSpace(m[2]))
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
			}
			ast.Matrices = append(ast.Matrices, MatrixLit{Name: m[1], Rows: rows})
			continue
		}

		if m := assignMatmulRe.FindStringSubmatch(line); m != nil {
			ast.Ops = append(ast.Ops, MatmulStmt{
				Left:  m[1],
				Right: m[2],
				Out:   m[3],
			})
			continue
		}

		if m := finalMatmulRe.FindStringSubmatch(line); m != nil {
			ast.Ops = append(ast.Ops, MatmulStmt{
				Left:  m[1],
				Right: m[2],
				Final: true,
			})
			continue
		}

		return nil, fmt.Errorf("%s:%d: could not parse line: %q", path, lineNo, line)
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return ast, nil
}

func readMatrixRows(sc *bufio.Scanner, lineNo *int, firstRest string) ([][]int64, error) {
	var rows [][]int64

	if firstRest != "" {
		done, row, err := parseMaybeMatrixEnd(firstRest)
		if err != nil {
			return nil, err
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
		if done {
			return validateRows(rows)
		}
	}

	for sc.Scan() {
		(*lineNo)++
		line := clean(sc.Text())
		if line == "" {
			continue
		}

		done, row, err := parseMaybeMatrixEnd(line)
		if err != nil {
			return nil, err
		}

		if len(row) > 0 {
			rows = append(rows, row)
		}

		if done {
			return validateRows(rows)
		}
	}

	return nil, errors.New("unterminated matrix literal")
}

func parseMaybeMatrixEnd(line string) (bool, []int64, error) {
	line = strings.TrimSpace(line)

	if line == "]" {
		return true, nil, nil
	}

	done := false
	if strings.HasSuffix(line, "]") {
		done = true
		line = strings.TrimSpace(strings.TrimSuffix(line, "]"))
	}

	if line == "" {
		return done, nil, nil
	}

	fields := strings.Fields(line)
	row := make([]int64, 0, len(fields))

	for _, f := range fields {
		x, err := strconv.ParseInt(f, 10, 64)
		if err != nil {
			return false, nil, fmt.Errorf("bad integer %q", f)
		}
		row = append(row, x)
	}

	return done, row, nil
}

func validateRows(rows [][]int64) ([][]int64, error) {
	if len(rows) == 0 {
		return nil, errors.New("empty matrix")
	}

	cols := len(rows[0])
	if cols == 0 {
		return nil, errors.New("matrix row has zero columns")
	}

	for i, r := range rows {
		if len(r) != cols {
			return nil, fmt.Errorf("ragged matrix: row 0 has %d cols, row %d has %d", cols, i, len(r))
		}
	}

	return rows, nil
}

func clean(s string) string {
	if idx := strings.Index(s, "#"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// ---------- compiler ----------

func (c *Compiler) compileFile(path string, isEntry bool) (*ModuleResult, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if existing, ok := c.Modules[path]; ok {
		return existing, nil
	}

	ast, err := parseFile(path)
	if err != nil {
		return nil, err
	}

	res := &ModuleResult{
		AST:     ast,
		Symbols: map[string]MatrixInfo{},
	}

	c.Modules[path] = res

	for _, imp := range ast.Imports {
		impPath := imp.From
		if !filepath.IsAbs(impPath) {
			impPath = filepath.Join(filepath.Dir(path), impPath)
		}

		dep, err := c.compileFile(impPath, false)
		if err != nil {
			return nil, err
		}

		info, ok := dep.Symbols[imp.Name]
		if !ok {
			return nil, fmt.Errorf("%s imports %q from %s, but it was not exported", path, imp.Name, imp.From)
		}

		res.Symbols[imp.Name] = info
	}

	m := ir.NewModule()

	// Declare putchar for the entry module.
	var putchar *ir.Func
	if isEntry {
		putchar = m.NewFunc("putchar", i32, ir.NewParam("ch", i32))
	}

	// Imported matrices are represented as external element functions.
	for _, imp := range ast.Imports {
		info := res.Symbols[imp.Name]
		declareMatrixElementFunctions(m, info)
	}

	// Matrix literals become exported element functions.
	for _, ml := range ast.Matrices {
		if _, exists := res.Symbols[ml.Name]; exists {
			return nil, fmt.Errorf("%s: duplicate symbol %q", path, ml.Name)
		}

		info := matrixInfoFromLiteral(path, ml)
		res.Symbols[ml.Name] = info
		defineMatrixElementFunctions(m, info)
	}

	var final *MatmulStmt

	for _, op := range ast.Ops {
		left, ok := res.Symbols[op.Left]
		if !ok {
			return nil, fmt.Errorf("%s: unknown matrix %q", path, op.Left)
		}

		right, ok := res.Symbols[op.Right]
		if !ok {
			return nil, fmt.Errorf("%s: unknown matrix %q", path, op.Right)
		}

		if left.Cols != right.Rows {
			return nil, fmt.Errorf(
				"%s: shape mismatch: %s is %dx%d, %s is %dx%d",
				path,
				op.Left, left.Rows, left.Cols,
				op.Right, right.Rows, right.Cols,
			)
		}

		if op.Final {
			tmp := op
			final = &tmp
			continue
		}

		values, err := matmulValues(left, right)
		if err != nil {
			return nil, err
		}

		outInfo := MatrixInfo{
			Name:       op.Out,
			SymbolBase: mangle(path, op.Out),
			Rows:       left.Rows,
			Cols:       right.Cols,
			Values:     values,
		}

		if _, exists := res.Symbols[op.Out]; exists {
			return nil, fmt.Errorf("%s: duplicate symbol %q", path, op.Out)
		}

		res.Symbols[op.Out] = outInfo
		defineMatrixElementFunctions(m, outInfo)
	}

	if isEntry {
		if final == nil {
			return nil, fmt.Errorf("%s: entry file needs a final expression like `w @ z`", path)
		}

		left := res.Symbols[final.Left]
		right := res.Symbols[final.Right]

		emitMain(m, putchar, left, right)
	}

	llPath := filepath.Join(c.BuildDir, mangleFile(path)+".ll")
	if err := os.WriteFile(llPath, []byte(m.String()), 0644); err != nil {
		return nil, err
	}

	res.LLPath = llPath
	c.Order = append(c.Order, path)

	return res, nil
}

func elemIndex(cols int, row int, col int) int {
	return row*cols + col
}

func validateMatrixInfo(info MatrixInfo) error {
	if info.Rows <= 0 || info.Cols <= 0 {
		return fmt.Errorf("%s: invalid matrix shape %dx%d", info.Name, info.Rows, info.Cols)
	}

	want := info.Rows * info.Cols
	if len(info.Values) != want {
		return fmt.Errorf(
			"%s: matrix data length mismatch: have %d values, want %d for %dx%d",
			info.Name,
			len(info.Values),
			want,
			info.Rows,
			info.Cols,
		)
	}

	return nil
}

func matrixInfoFromLiteral(path string, ml MatrixLit) MatrixInfo {
	rows := len(ml.Rows)
	cols := len(ml.Rows[0])

	var values []int64
	for _, r := range ml.Rows {
		values = append(values, r...)
	}

	return MatrixInfo{
		Name:       ml.Name,
		SymbolBase: mangle(path, ml.Name),
		Rows:       rows,
		Cols:       cols,
		Values:     values,
	}
}

func matmulValues(a, b MatrixInfo) ([]int64, error) {
	if err := validateMatrixInfo(a); err != nil {
		return nil, err
	}
	if err := validateMatrixInfo(b); err != nil {
		return nil, err
	}

	if a.Cols != b.Rows {
		return nil, fmt.Errorf(
			"shape mismatch: %s is %dx%d, %s is %dx%d",
			a.Name,
			a.Rows, a.Cols,
			b.Name,
			b.Rows, b.Cols,
		)
	}

	outRows := a.Rows
	outCols := b.Cols
	out := make([]int64, outRows*outCols)

	for r := 0; r < outRows; r++ {
		for c := 0; c < outCols; c++ {
			var sum int64

			for k := 0; k < a.Cols; k++ {
				// Row-major indexing:
				//   a[r][k]   => a.Values[r*a.Cols+k]
				//   b[k][c]   => b.Values[k*b.Cols+c]
				//   out[r][c] => out[r*outCols+c]
				av := a.Values[elemIndex(a.Cols, r, k)]
				bv := b.Values[elemIndex(b.Cols, k, c)]

				sum += av * bv
			}

			out[elemIndex(outCols, r, c)] = sum
		}
	}

	return out, nil
}

// ---------- LLVM function exports instead of globals ----------

func elemFuncName(info MatrixInfo, row int, col int) string {
	return fmt.Sprintf("%s_%d_%d", info.SymbolBase, row, col)
}

func defineMatrixElementFunctions(m *ir.Module, info MatrixInfo) {
	if err := validateMatrixInfo(info); err != nil {
		panic(err)
	}

	for r := 0; r < info.Rows; r++ {
		for c := 0; c < info.Cols; c++ {
			idx := elemIndex(info.Cols, r, c)
			name := elemFuncName(info, r, c)

			fn := m.NewFunc(name, i32)
			entry := fn.NewBlock("entry")
			entry.NewRet(constant.NewInt(i32, info.Values[idx]))
		}
	}
}

func declareMatrixElementFunctions(m *ir.Module, info MatrixInfo) {
	for r := 0; r < info.Rows; r++ {
		for c := 0; c < info.Cols; c++ {
			m.NewFunc(elemFuncName(info, r, c), i32)
		}
	}
}

// ---------- main generation ----------

func emitMain(m *ir.Module, putchar *ir.Func, left MatrixInfo, right MatrixInfo) {
	mainFn := m.NewFunc("main", i32)
	entry := mainFn.NewBlock("entry")

	result, err := matmulValues(left, right)
	if err != nil {
		panic(err)
	}

	var text strings.Builder
	text.WriteString("Result:\n")

	for r := 0; r < left.Rows; r++ {
		for c := 0; c < right.Cols; c++ {
			if c > 0 {
				text.WriteString(" ")
			}
			text.WriteString(strconv.FormatInt(result[elemIndex(right.Cols, r, c)], 10))
		}
		text.WriteString("\n")
	}

	emitPutString(entry, putchar, text.String())
	entry.NewRet(constant.NewInt(i32, 0))
}

func emitPutString(block *ir.Block, putchar *ir.Func, s string) {
	for _, ch := range []byte(s) {
		block.NewCall(putchar, constant.NewInt(i32, int64(ch)))
	}
}

// ---------- names ----------

func mangle(path string, name string) string {
	return mangleFile(path) + "_" + name
}

func mangleFile(path string) string {
	abs, _ := filepath.Abs(path)
	s := strings.ToLower(abs)

	var b strings.Builder
	b.WriteString("mat_")

	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	return b.String()
}
