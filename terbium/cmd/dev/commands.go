package main

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/Awesome-Sauces/terbium/internal/driver"
	"github.com/Awesome-Sauces/terbium/internal/log"
)

var (
	verbose bool
	trace   bool
)

var rootCmd = &cobra.Command{
	Use:   "terbium",
	Short: "Terbium is the compiler for the Terb programming language.",
	Long:  `terbium dev build`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configureLogger()
	},
}

var emitLLVMCmd = &cobra.Command{
	Use:   "emit-llvm <file.terb>",
	Short: "Emit LLVM IR from a Terb source file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputPath := args[0]

		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		log.Debug(
			"Resolved emit-llvm paths",
			"input", inputPath,
			"output", outputPath,
		)

		log.Info(
			"Emitting LLVM IR",
			"input", inputPath,
			"output", outputPath,
		)

		if err := driver.EmitLLVM(inputPath, outputPath); err != nil {
			log.Fatal(err)
		}

		log.Info(
			"Wrote LLVM IR",
			"output", outputPath,
		)
	},
}

var buildCmd = &cobra.Command{
	Use:   "build <file.terb>",
	Short: "Build a Terb source file into an executable",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputPath := args[0]

		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		log.Debug(
			"Resolved build paths",
			"input", inputPath,
			"output", outputPath,
		)

		log.Info(
			"Building executable",
			"input", inputPath,
			"output", outputPath,
		)

		if err := driver.Build(inputPath, outputPath); err != nil {
			log.Fatal(err)
		}

		log.Info(
			"Built executable",
			"output", outputPath,
		)
	},
}

var testLexer = &cobra.Command{
	Use:   "lexer <file.terb>",
	Short: "Build a Terb source file into an executable",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputPath := args[0]

		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		log.Debug(
			"Testing Lexer with",
			"file", inputPath,
		)

		driver.Dev5262026_Lexer(inputPath, outputPath)

		log.Info(
			"Built executable",
			"output", outputPath,
		)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"enable debug logging",
	)

	rootCmd.PersistentFlags().BoolVar(
		&trace,
		"trace",
		false,
		"enable trace logging",
	)

	emitLLVMCmd.Flags().StringP(
		"output",
		"o",
		"out.ll",
		"output LLVM IR file",
	)

	buildCmd.Flags().StringP(
		"output",
		"o",
		"out",
		"output executable path",
	)

	testLexer.Flags().StringP(
		"output",
		"o",
		"out",
		"output executable path",
	)

	rootCmd.AddCommand(emitLLVMCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(testLexer)
}

func CommandsExecute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func configureLogger() {
	switch {
	case trace:
		log.SetLevel(pterm.LogLevelTrace)

	case verbose:
		log.SetLevel(pterm.LogLevelDebug)

	default:
		log.SetLevel(pterm.LogLevelInfo)
	}
}
