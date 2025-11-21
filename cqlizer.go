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
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cqlizer/apiserver"
	"github.com/czcorpus/cqlizer/cnf"
)

const (
	actionMCPServer        = "mcp-server"
	actionREPL             = "repl"
	actionVersion          = "version"
	actionHelp             = "help"
	actionLearn            = "learn"
	actionFeaturize        = "featurize"
	actionEvaluate         = "evaluate"
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

//go:embed scripts/rfchart.py
var rfChartScript string

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
	fmt.Fprintf(os.Stderr, "\t%s\t\t\tlearn model based on provided features\n", actionLearn)
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

	cmdKlogImport := flag.NewFlagSet(actionLearn, flag.ExitOnError)
	numTrees := cmdKlogImport.Int("num-trees", 100, "Number of trees for Random Forest (default: 100)")
	klogImportModel := cmdKlogImport.String("model", "rf", "Specifies model which will be used (nn, rf)")
	voteThreshold := cmdKlogImport.Float64("vote-threshold", 0, "RF Vote threshold for marking CQL as problematic. This affects only evaluation. If none, then range from 0.7 to 0.99 is examined")
	klogImportMisclassOut := cmdKlogImport.String("misclassed-query-log", "", "Specify a path to store misclassified queries. If none, no logging is performed.")
	klogImportForXBGoost := cmdKlogImport.Bool(
		"for-xgboost",
		false,
		"if set then CQLizer will export a file for external XGBoost model learning",
	)
	cmdKlogImport.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s learn [options] config.json features_file.msgpack\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdKlogImport.PrintDefaults()
	}

	cmdEvaluate := flag.NewFlagSet(actionEvaluate, flag.ExitOnError)
	cmdEvaluateModel := cmdEvaluate.String("model", "rf", "Specifies model which will be used (nn, rf)")
	cmdEvaluateMisclassOut := cmdEvaluate.String("misclassed-query-log", "", "Specify a path to store misclassified queries. If none, no logging is performed.")
	cmdEvaluate.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s evaluate [options] config.json model_file testing_data \n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdEvaluate.PrintDefaults()
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
	benchmarkSpecCorpora := cmdBenchmarkMissing.String("corpora", "", "A forced list of comma-separated corpora to process, everything else ignored. If not set, all the corpora found in MQuery will be used.")
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
		case actionLearn:
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
	case actionLearn:
		cmdKlogImport.Parse(os.Args[2:])
		conf := setup(cmdKlogImport.Arg(0))

		runActionKlogImport(
			conf,
			cmdKlogImport.Arg(1),
			*klogImportModel,
			*numTrees,
			*voteThreshold,
			*klogImportMisclassOut,
			*klogImportForXBGoost,
		)
	case actionEvaluate:
		cmdEvaluate.Parse(os.Args[2:])
		conf := setup(cmdEvaluate.Arg(0))
		runActionEvaluate(
			conf,
			cmdEvaluate.Arg(1),
			*cmdEvaluateModel,
			cmdEvaluate.Arg(2),
			*cmdEvaluateMisclassOut,
		)

	case actionFeaturize:
		cmdFeaturize.Parse(os.Args[2:])
		conf := setup(cmdFeaturize.Arg(0))
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		runActionFeaturize(
			ctx,
			conf,
			cmdFeaturize.Arg(1),
			cmdFeaturize.Arg(2),
			*featurizeDebug,
		)
	case actionBenchmarkMissing:
		cmdBenchmarkMissing.Parse(os.Args[2:])
		conf := setup(cmdBenchmarkMissing.Arg(0))
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		corpora := []string{}
		if *benchmarkSpecCorpora != "" {
			corpora = strings.Split(*benchmarkSpecCorpora, ",")
		}
		runActionBenchmarkMissing(
			ctx,
			conf,
			cmdBenchmarkMissing.Arg(1),
			corpora,
			*benchmarkBatchSize,
			*benchmarkBatchOffset,
		)
	case actionRemoveZero:
		cmdRemoveZero.Parse(os.Args[2:])
		conf := setup(cmdRemoveZero.Arg(0))
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		runActionRemoveZero(
			ctx,
			conf,
			cmdRemoveZero.Arg(1),
		)

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
