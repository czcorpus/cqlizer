// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
// Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate pigeon -o ./cql/grammar.go ./cql/grammar.peg

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/dataimport"
	"github.com/czcorpus/cqlizer/eval"
)

const (
	actionMCPServer  = "mcp-server"
	actionREPL       = "repl"
	actionVersion    = "version"
	actionHelp       = "help"
	actionKlogImport = "klog-import"

	exitErrorGeneralFailure = iota
	exitErrorImportFailed
	exiterrrorREPLReading
	exitErrorFailedToOpenIdex
	exitErrorFailedToOpenQueryPersistence
	exitErrorFailedToOpenW2VModel
)

var (
	version   string
	buildDate string
	gitCommit string
)

// VersionInfo provides a detailed information about the actual build
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
}

func topLevelUsage() {
	fmt.Fprintf(os.Stderr, "CQLIZER - a data-driven CQL writing helper tool\n")
	fmt.Fprintf(os.Stderr, "-----------------------------\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "\t%s\t\t\tshow version info\n", actionVersion)
	fmt.Fprintf(os.Stderr, "\t%s\t\tmcp-server MCP \n", actionMCPServer)
	fmt.Fprintf(os.Stderr, "\t%s\t\t\trepl \n", actionREPL)
	fmt.Fprintf(os.Stderr, "\t%s\t\t\tklog-import \n", actionKlogImport)
	fmt.Fprintf(os.Stderr, "\nUse `cqlizer help ACTION` for information about a specific action\n\n")
}

func setup(confPath string) *cnf.Conf {
	conf := cnf.LoadConfig(confPath)
	if conf.Logging.Level == "" {
		conf.Logging.Level = "info"
	}
	logging.SetupLogging(conf.Logging)
	cnf.ValidateAndDefaults(conf)
	return conf
}

func cleanVersionInfo(v string) string {
	return strings.TrimLeft(strings.Trim(v, "'"), "v")
}

func runActionMCPServer() {

}

func runActionREPL(modelPath string) {
	// Load the model
	fmt.Printf("Loading model from %s...\n", modelPath)
	model, err := eval.LoadModelFromFile(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading model: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Model loaded successfully\n")

	// Default corpus size (can be overridden with 'set corpussize <value>')
	corpusSize := 100000000.0 // 100M tokens default

	fmt.Println("CQL Query Cost Estimator")
	fmt.Println("Commands:")
	fmt.Println("  <CQL query>            - Estimate query execution time")
	fmt.Println("  set corpussize <size>  - Set corpus size (e.g., 'set corpussize 121826797')")
	fmt.Println("  exit                   - Exit REPL")
	fmt.Printf("\nCurrent corpus size: %.0f tokens\n\n", corpusSize)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Handle 'set corpussize' command
		if strings.HasPrefix(input, "set corpussize ") {
			parts := strings.Fields(input)
			if len(parts) == 3 {
				var newSize float64
				if _, err := fmt.Sscanf(parts[2], "%f", &newSize); err == nil {
					corpusSize = newSize
					fmt.Printf("✓ Corpus size set to %.0f tokens\n", corpusSize)
				} else {
					fmt.Fprintf(os.Stderr, "Error: Invalid corpus size\n")
				}
			} else {
				fmt.Fprintf(os.Stderr, "Usage: set corpussize <size>\n")
			}
			continue
		}

		// Treat as CQL query
		queryEval, err := eval.NewQueryEvaluation(input, corpusSize, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing CQL: %v\n", err)
			continue
		}

		// Calculate estimated cost
		estimatedTime := queryEval.Cost(model)

		// Display results
		fmt.Printf("\n--- Query Analysis ---\n")
		fmt.Printf("Query:           %s\n", input)
		fmt.Printf("Corpus size:     %.0f tokens\n", corpusSize)
		fmt.Printf("Positions:       %d\n", len(queryEval.Positions))
		for i, pos := range queryEval.Positions {
			fmt.Printf("  Position %d:    wildcards=%d, range=%d, smallCard=%d, numConcreteChars=%d\n",
				i, pos.Regexp.NumWildcards, pos.Regexp.HasRange, pos.HasSmallCardAttr, pos.Regexp.NumConcreteChars)
		}
		fmt.Printf("Global features: glob=%d, meet=%d, union=%d, within=%d, containing=%d\n",
			queryEval.NumGlobConditions, queryEval.ContainsMeet,
			queryEval.ContainsUnion, queryEval.ContainsWithin, queryEval.ContainsContaining)
		fmt.Printf("\n⏱️  Estimated time: %.4f seconds\n\n", estimatedTime)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

func runActionKlogImport(conf *cnf.Conf, srcPath string, useRF bool, numTrees int, voteThreshold float64) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fmt.Println("ctx: ", ctx)

	model := &eval.BasicModel{}
	dataimport.ReadStatsFile(srcPath, model)
	allEvals := model.BalanceSample()

	if useRF {
		// Train Random Forest model
		if err := model.EvaluateWithRF(numTrees, voteThreshold, allEvals, ""); err != nil {
			fmt.Fprintf(os.Stderr, "RF training failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Train Huber regression model (default)
		model.Evaluate()
	}
}

func runActionVersion(ver VersionInfo) {
	fmt.Fprintln(os.Stderr, "CQLizer version: ", ver)
}

func main() {
	version := VersionInfo{
		Version:   cleanVersionInfo(version),
		BuildDate: cleanVersionInfo(buildDate),
		GitCommit: cleanVersionInfo(gitCommit),
	}

	cmdMCP := flag.NewFlagSet(actionMCPServer, flag.ExitOnError)
	cmdMCP.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json\n\t",
			filepath.Base(os.Args[0]), actionMCPServer)
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdMCP.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSrun CQLizer as a MCP server\n")
	}

	cmdVersion := flag.NewFlagSet(actionVersion, flag.ExitOnError)
	cmdVersion.Usage = func() {
		cmdVersion.PrintDefaults()
		// TOOD
	}

	cmdHelp := flag.NewFlagSet(actionHelp, flag.ExitOnError)
	cmdHelp.Usage = func() {
		cmdVersion.PrintDefaults()
	}

	cmdREPL := flag.NewFlagSet(actionREPL, flag.ExitOnError)
	cmdREPL.Usage = func() {
		cmdREPL.PrintDefaults()
	}

	cmdKlogImport := flag.NewFlagSet(actionKlogImport, flag.ExitOnError)
	useRF := cmdKlogImport.Bool("rf", false, "Use Random Forest instead of Huber regression")
	numTrees := cmdKlogImport.Int("trees", 100, "Number of trees for Random Forest (default: 100)")
	voteThreshold := cmdKlogImport.Float64("vote-threshold", 0.3, "RF Vote threshold for marking CQL as problematic")
	cmdKlogImport.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s klog-import [options] config.json logfile.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdKlogImport.PrintDefaults()
	}

	action := actionHelp
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	switch action {
	case actionHelp:
		var subj string
		if len(os.Args) > 2 {
			cmdHelp.Parse(os.Args[2:])
			subj = cmdHelp.Arg(0)
		}
		if subj == "" {
			topLevelUsage()
			return
		}
		switch subj {
		case actionKlogImport:
			cmdKlogImport.PrintDefaults()
		case actionMCPServer:
			cmdMCP.PrintDefaults()
		case actionREPL:
			cmdREPL.PrintDefaults()
		}
	case actionVersion:
		cmdVersion.Parse(os.Args[2:])
		runActionVersion(version)
	case actionMCPServer:
		cmdMCP.Parse(os.Args[2:])
		runActionMCPServer()
	case actionREPL:
		cmdREPL.Parse(os.Args[2:])
		if cmdREPL.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Error: model file path required\n")
			fmt.Fprintf(os.Stderr, "Usage: %s repl <model_file.json>\n", os.Args[0])
			os.Exit(1)
		}
		modelPath := cmdREPL.Arg(0)
		runActionREPL(modelPath)
	case actionKlogImport:
		cmdKlogImport.Parse(os.Args[2:])
		conf := setup(cmdKlogImport.Arg(0))

		runActionKlogImport(conf, cmdKlogImport.Arg(1), *useRF, *numTrees, *voteThreshold)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action, please use 'help' to get more information")
	}

}
