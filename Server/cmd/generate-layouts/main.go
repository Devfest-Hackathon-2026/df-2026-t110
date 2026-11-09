// Command generate-layouts creates 64 symmetric battle boards offline.
//
// From the repository root:
//
//	go -C Server run ./cmd/generate-layouts --output ../work/layouts.json
//
// Use --output infra/matchrun/layouts.json to replace the embedded catalog.
package main

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("generate-layouts", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Uint64("seed", 165, "random seed")
	output := flags.String("output", "", "destination JSON file (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" || flags.NArg() != 0 {
		return errors.New("usage: generate-layouts --output path/to/layouts.json [--seed 165]")
	}
	layouts, err := generate(*seed)
	if err != nil {
		return err
	}
	data, err := json.Marshal(layouts, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode layouts: %w", err)
	}
	// Finish generation before opening the destination to avoid truncating it
	// when generation or encoding fails.
	if err := os.MkdirAll(filepath.Dir(*output), 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(*output, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write layouts: %w", err)
	}
	return nil
}
