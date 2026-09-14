package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/holdout-bench/holdout/internal/catalog"
	"github.com/holdout-bench/holdout/internal/model"
	"github.com/holdout-bench/holdout/internal/runner"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	switch args[0] {
	case "list":
		return listCommand(root, args[1:])
	case "validate":
		return validateCommand(root, args[1:])
	case "validate-result":
		return validateResultCommand(args[1:])
	case "run":
		return runCommand(root, args[1:])
	default:
		usage()
		return 2
	}
}

func listCommand(root string, args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "suts":
		fmt.Println("docker-runc\tDocker Engine/runc baseline")
		fmt.Println("docker\talias for docker-runc")
		fmt.Println("gvisor\tDocker Engine with runsc runtime")
		fmt.Println("opensandbox\tplanned adapter")
		return 0
	case "tasks":
		fs := flag.NewFlagSet("list tasks", flag.ContinueOnError)
		suite := fs.String("suite", "smoke-v0", "suite name")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		tasks, err := catalog.Load(filepath.Join(root, "tasks", "catalog.yaml"), *suite)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		for _, task := range tasks {
			fmt.Printf("%s\t%s\t%s\t%s\n", task.ID, task.Family, task.Level, task.Title)
		}
		return 0
	default:
		usage()
		return 2
	}
}

func validateCommand(root string, args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	suite := fs.String("suite", "smoke-v0", "suite name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	tasks, err := catalog.Load(filepath.Join(root, "tasks", "catalog.yaml"), *suite)
	if err == nil {
		err = catalog.Validate(root, tasks)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	fmt.Printf("valid: %d tasks (%s)\n", len(tasks), *suite)
	return 0
}

func validateResultCommand(args []string) int {
	fs := flag.NewFlagSet("validate-result", flag.ContinueOnError)
	file := fs.String("file", "results.json", "result JSON path")
	strict := fs.Bool("strict", false, "require exact catalog task coverage and digests")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	f, err := os.Open(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var result model.RunResult
	if err := decoder.Decode(&result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			fmt.Fprintln(os.Stderr, "result file contains more than one JSON value")
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		return 2
	}
	if err := model.ValidateRunResult(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *strict {
		root, err := findRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if err := catalog.ValidateResult(root, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	fmt.Printf("valid result: %s (%s, %d tasks)\n", result.RunID, result.Suite, len(result.Tasks))
	return 0
}

func runCommand(root string, args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	sutName := fs.String("sut", "docker-runc", "SUT adapter")
	suite := fs.String("suite", "smoke-v0", "suite name")
	out := fs.String("out", "results.json", "result output path")
	image := fs.String("image", "", "preloaded Docker image (required for real runs)")
	allowDangerous := fs.Bool("allow-dangerous", false, "allow dangerous tasks in an isolated outer VM")
	dryRun := fs.Bool("dry-run", false, "validate and emit skipped task results without Docker")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *allowDangerous && os.Getenv("HOLDOUT_I_UNDERSTAND") != "1" {
		fmt.Fprintln(os.Stderr, "--allow-dangerous requires HOLDOUT_I_UNDERSTAND=1")
		return 2
	}
	result, err := runner.Run(context.Background(), runner.Options{Root: root, Suite: *suite, SUT: *sutName, Image: *image, AllowDangerous: *allowDangerous, DryRun: *dryRun})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if result.RunID != "" {
			if writeErr := runner.Write(*out, result); writeErr != nil {
				fmt.Fprintln(os.Stderr, writeErr)
			}
		}
		return 2
	}
	if err := runner.Write(*out, result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	data, _ := json.Marshal(result.Summary)
	fmt.Printf("run %s -> %s\nsummary: %s\n", result.RunID, *out, data)
	for _, task := range result.Tasks {
		if task.Status == "error" || task.Status == "skipped" || task.CleanupError != "" {
			return 3
		}
		if task.Status == "bypass" || task.Status == "exfil" || task.Status == "lateral" || task.Status == "escape" {
			return 1
		}
	}
	return 0
}

func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "tasks", "catalog.yaml")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("holdout repository root not found from %s", dir)
		}
		dir = parent
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: holdout list suts|tasks [--suite NAME] | validate [--suite NAME] | validate-result --file PATH | run [flags]")
}
