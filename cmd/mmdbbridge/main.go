package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ipanalytics/mmdbbridge/internal/bridge"
	"github.com/ipanalytics/mmdbbridge/internal/schema"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mmdbbridge:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("missing command")
	}

	switch args[0] {
	case "build":
		fs := flag.NewFlagSet("build", flag.ContinueOnError)
		schemaPath := fs.String("schema", "", "YAML schema path")
		csvPath := fs.String("csv", "", "input CSV path")
		outPath := fs.String("out", "", "output MMDB path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *schemaPath == "" || *csvPath == "" || *outPath == "" {
			return fmt.Errorf("build requires --schema, --csv, and --out")
		}
		s, err := schema.Load(*schemaPath)
		if err != nil {
			return err
		}
		return bridge.Build(s, *csvPath, *outPath)

	case "export":
		fs := flag.NewFlagSet("export", flag.ContinueOnError)
		schemaPath := fs.String("schema", "", "YAML schema path")
		mmdbPath := fs.String("mmdb", "", "input MMDB path")
		outPath := fs.String("out", "", "output CSV path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *schemaPath == "" || *mmdbPath == "" || *outPath == "" {
			return fmt.Errorf("export requires --schema, --mmdb, and --out")
		}
		s, err := schema.Load(*schemaPath)
		if err != nil {
			return err
		}
		return bridge.Export(s, *mmdbPath, *outPath)

	case "schema":
		fs := flag.NewFlagSet("schema", flag.ContinueOnError)
		outPath := fs.String("out", "", "output schema path; stdout when empty")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *outPath == "" {
			fmt.Print(schema.Example)
			return nil
		}
		return os.WriteFile(*outPath, []byte(schema.Example), 0o644)

	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `MMDBridge: typed CSV <-> MMDB bridge

Usage:
  mmdbbridge build  --schema schema.yaml --csv input.csv --out output.mmdb
  mmdbbridge export --schema schema.yaml --mmdb input.mmdb --out output.csv
  mmdbbridge schema [--out schema.yaml]`)
}
