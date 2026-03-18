package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/farmcan/canx/internal/evalreport"
)

func main() {
	var (
		real   = flag.Bool("real", false, "run real Codex evals")
		outDir = flag.String("out-dir", "evals/reports", "directory for generated reports")
	)
	flag.Parse()

	args := []string{"test", "./evals/agentic", "-json"}
	if *real {
		args = append(args, "-run", "TestAgenticRealExecSmokeIfEnabled|TestPlannerRealSmokeIfEnabled")
	} else {
		args = append(args, "-run", "TestAgenticQuickSuite")
	}

	cmd := exec.Command("go", args...)
	if *real {
		cmd.Env = append(os.Environ(), "CANX_EVAL_REAL=1")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: %v\n%s", err, string(output))
		os.Exit(1)
	}

	report, err := evalreport.ParseGoTestJSON(bytes.NewReader(output))
	if err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: parse report: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: mkdir: %v\n", err)
		os.Exit(1)
	}

	rawPath := filepath.Join(*outDir, "latest.jsonl")
	jsonPath := filepath.Join(*outDir, "latest.json")
	mdPath := filepath.Join(*outDir, "latest.md")
	if err := os.WriteFile(rawPath, output, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: write raw: %v\n", err)
		os.Exit(1)
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: marshal report: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(jsonPath, payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: write json: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(mdPath, []byte(evalreport.RenderMarkdown(report)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "canx-eval-report: write markdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("report written: %s %s %s\n", rawPath, jsonPath, mdPath)
}
