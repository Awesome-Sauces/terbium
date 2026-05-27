package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Awesome-Sauces/terbium/internal/codegen"
	"github.com/Awesome-Sauces/terbium/internal/lexer"
	"github.com/Awesome-Sauces/terbium/internal/log"
	"github.com/Awesome-Sauces/terbium/internal/parser"
)

func EmitLLVM(inputPath string, outputPath string) error {
	src, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	program, err := parser.Parse(string(src))
	if err != nil {
		return err
	}

	irText, err := codegen.Generate(program)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, []byte(irText), 0644)
}

func Build(inputPath string, outputPath string) error {

	log.Trace("Could not build properly")

	llPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".ll"

	if err := EmitLLVM(inputPath, llPath); err != nil {
		return err
	}

	cmd := exec.Command("clang", llPath, "-o", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clang failed: %w", err)
	}

	return nil
}

// Temporary code while implementing Lexer

func Dev5262026_EmitLexical(inputPath string, outputPath string) error {

	src, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	log.Info("Lexing source", "file", inputPath, "filesize", len(src))

	tokens, err := lexer.Lex(string(src))
	if err != nil {
		return err
	}

	log.Info("Generated tokens", "count", len(tokens))

	return nil
}

func Dev5262026_Lexer(inputPath string, outputPath string) error {

	if err := Dev5262026_EmitLexical(inputPath, outputPath); err != nil {
		return err
	}

	return nil
}
