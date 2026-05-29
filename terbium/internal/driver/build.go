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

	tokens, err := lexer.Lex(inputPath)
	if err != nil {
		return err
	}

	for i := range tokens {
		info_args := []string{}

		if tokens[i].Children != nil {
			for x := range tokens[i].Children {
				if tokens[i].Children[x].Children != nil {
					info_args = append(info_args, fmt.Sprintf("%s (contains %d children)", lexer.TokenTypeToString(tokens[i].Children[x].Type), len(tokens[i].Children[x].Children)))
				} else {
					info_args = append(info_args, lexer.TokenTypeToString(tokens[i].Children[x].Type))
				}
				info_args = append(info_args, string(tokens[i].Children[x].Literal))
			}
		}

		log.Info(lexer.TokenTypeToString(tokens[i].Type), info_args)
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
