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
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cqlizer/apiserver"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/nn"
	"github.com/czcorpus/cqlizer/eval/rf"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	actionMCPServer        = "mcp-server"
	actionREPL             = "repl"
	actionVersion          = "version"
	actionHelp             = "help"
	actionKlogImport       = "klog-import"
	actionFeaturize        = "featurize"
	actionBenchmarkMissing = "benchmark-missing"
	actionRemoveZero       = "remove-zero"
	actionAPIServer        = "server"

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
	fmt.Fprintf(os.Stderr, "\t%s\t\t\ttransform query log into features\n", actionFeaturize)
	fmt.Fprintf(os.Stderr, "\t%s\t\t\tlearn model based on provided features\n", actionKlogImport)
	fmt.Fprintf(os.Stderr, "\t%s\t\t\tbenchmark queries with zero processing time (using MQuery)\n", actionBenchmarkMissing)
	fmt.Fprintf(os.Stderr, "\t%s\t\t\tREPL model \n", actionREPL)
	fmt.Fprintf(os.Stderr, "\t%s\t\tmcp-server MCP (experimental) \n", actionMCPServer)
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

func runActionREPL(rfPath string) {
	var rfModel *rf.Model
	var err error
	if rfPath != "" {
		rfModel, err = rf.LoadFromFile(rfPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading RF model: %v\n", err)
			os.Exit(1)
		}

	} else {
		fmt.Println("no RF model specified")
	}

	// Default corpus size (can be overridden with 'set corpussize <value>')
	corpusSize := 6400000000.0 // 6.4G tokens default

	lang := "cs"

	fmt.Println("CQL Query Cost Estimator")
	fmt.Println("Commands:")
	fmt.Println("  <CQL query>            - Estimate query execution time")
	fmt.Println("  set corpussize <size>  - Set corpus size (e.g., 'set corpussize 121826797')")
	fmt.Println("  set lang <lang>  - Set corpus language (e.g., 'set lang cs')")
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

		} else if strings.HasPrefix(input, "set lang ") {
			parts := strings.Fields(input)
			if len(parts) == 3 {
				lang = parts[2]

			} else {
				fmt.Fprintf(os.Stderr, "Usage: set lang <lang>\n")
			}
			continue
		}

		// Treat as CQL query
		charProbs := feats.GetCharProbabilityProvider(lang)
		queryEval, err := feats.NewQueryEvaluation(input, corpusSize, 0, 0, charProbs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing CQL: %v\n", err)
			continue
		}

		// Display results
		fmt.Printf("\n--- Query Analysis ---\n")
		fmt.Printf("Query:           %s\n", input)
		fmt.Printf("Corpus size:     %.0f tokens\n", corpusSize)
		fmt.Printf("Positions:       %d\n", len(queryEval.Positions))
		for i, pos := range queryEval.Positions {
			fmt.Printf("  Position %d:    wildcards=%0.2f, range=%d, smallCard=%d, numConcreteChars=%.2f, posNumAlts: %d\n",
				i, pos.Regexp.WildcardScore, pos.Regexp.HasRange, pos.HasSmallCardAttr, pos.Regexp.NumConcreteChars, pos.NumAlternatives)
		}
		fmt.Printf("Global features: glob=%d, meet=%d, union=%d, within=%d, containing=%d\n",
			queryEval.NumGlobConditions, queryEval.ContainsMeet,
			queryEval.ContainsUnion, queryEval.ContainsWithin, queryEval.ContainsContaining)

		if rfModel != nil {
			rfPRediction := rfModel.Predict(queryEval)
			fmt.Printf("RF prediction: %d\n", rfPRediction.PredictedClass)
			fmt.Printf("votes: %#v\n", rfPRediction.Votes)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

func runActionKlogImport(conf *cnf.Conf, srcPath string, modelType string, numTrees int, voteThreshold float64) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	/*
		model := &eval.BasicModel{
			SlowQueryPercentile: slowQueryPerc,
		}
		dataimport.ReadStatsFile(ctx, srcPath, model)
	*/
	srcPathExt := filepath.Ext(srcPath)
	outFile := fmt.Sprintf("%s.%s.json", srcPath[:len(srcPath)-len(srcPathExt)], modelType)
	f, err := os.Open(srcPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}

	var mlModel eval.MLModel
	switch modelType {
	case "rf":
		mlModel = rf.NewModel(numTrees, voteThreshold)
	case "nn":
		mlModel = nn.NewModel()
	default:
		log.Fatal().Str("modelType", modelType).Msg("Unknown model")
		return
	}

	model := eval.NewPredictor(mlModel, conf.CorporaProps)
	if err := msgpack.Unmarshal(data, &model); err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}

	allEvals := model.BalanceSample()

	if err := model.CreateAndTestModel(ctx, allEvals, outFile); err != nil {
		fmt.Fprintf(os.Stderr, "RF training failed: %v\n", err)
		os.Exit(1)
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
	numTrees := cmdKlogImport.Int("num-trees", 100, "Number of trees for Random Forest (default: 100)")
	klogImportModel := cmdKlogImport.String("model", "rf", "Specifies model which will be used (nn, rf)")
	voteThreshold := cmdKlogImport.Float64("vote-threshold", 0, "RF Vote threshold for marking CQL as problematic. This affects only evaluation. If none, then range from 0.7 to 0.99 is examined")
	cmdKlogImport.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s klog-import [options] config.json logfile.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdKlogImport.PrintDefaults()
	}

	cmdFeaturize := flag.NewFlagSet(actionFeaturize, flag.ExitOnError)
	featurizeDebug := cmdFeaturize.Bool(
		"debug",
		false,
		"if set then features will be written to stdout in human readable form and no feats file will be created",
	)
	cmdFeaturize.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s featurize [options] config.json logfile.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdFeaturize.PrintDefaults()
	}

	cmdBenchmarkMissing := flag.NewFlagSet(actionBenchmarkMissing, flag.ExitOnError)
	benchmarkSpecCorpora := cmdBenchmarkMissing.String("corpora", "", "A list of comma-separated corpora to process, everything else ignored")
	benchmarkBatchSize := cmdBenchmarkMissing.Int("batch-size", 0, "Max. number of items to process at once")
	benchmarkBatchOffset := cmdBenchmarkMissing.Int("batch-offset", 0, "Where (in the sorted list of entries; zero indexed) to start with the current run.")
	cmdBenchmarkMissing.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s benchmark-missing logfile.jsonl mquery_url\n", os.Args[0])
		cmdBenchmarkMissing.PrintDefaults()
	}

	cmdRemoveZero := flag.NewFlagSet(actionRemoveZero, flag.ExitOnError)
	cmdRemoveZero.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s remove-zero logfile.jsonl\n", os.Args[0])
		cmdRemoveZero.PrintDefaults()
	}

	cmdAPIServer := flag.NewFlagSet(actionAPIServer, flag.ExitOnError)
	cmdAPIServer.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s server [options] config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdAPIServer.PrintDefaults()
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
		fmt.Println("MODEL: ", modelPath)
		runActionREPL(modelPath)
	case actionKlogImport:
		cmdKlogImport.Parse(os.Args[2:])
		conf := setup(cmdKlogImport.Arg(0))

		runActionKlogImport(
			conf,
			cmdKlogImport.Arg(1),
			*klogImportModel,
			*numTrees,
			*voteThreshold,
		)
	case actionFeaturize:
		cmdFeaturize.Parse(os.Args[2:])
		conf := setup(cmdFeaturize.Arg(0))
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		runActionFeaturize(
			ctx,
			conf.CorporaProps,
			cmdFeaturize.Arg(1),
			cmdFeaturize.Arg(2),
			*featurizeDebug,
		)
	case actionBenchmarkMissing:
		cmdBenchmarkMissing.Parse(os.Args[2:])
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		corpora := []string{}
		if *benchmarkSpecCorpora != "" {
			corpora = strings.Split(*benchmarkSpecCorpora, ",")
		}
		runActionBenchmarkMissing(
			ctx,
			cmdBenchmarkMissing.Arg(0),
			cmdBenchmarkMissing.Arg(1),
			corpora,
			*benchmarkBatchSize,
			*benchmarkBatchOffset,
		)
	case actionRemoveZero:
		cmdBenchmarkMissing.Parse(os.Args[2:])
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		runActionRemoveZero(ctx, cmdBenchmarkMissing.Arg(0))

	case actionAPIServer:
		cmdAPIServer.Parse(os.Args[2:])
		conf := setup(cmdAPIServer.Arg(0))
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		apiserver.Run(ctx, conf)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action, please use 'help' to get more information")
	}

}
