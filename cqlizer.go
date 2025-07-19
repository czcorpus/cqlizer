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
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/dataimport"
	"github.com/czcorpus/cqlizer/embedding"
	"github.com/czcorpus/cqlizer/index"
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

func runActionREPL(db *index.DB, model *embedding.CQLEmbedder) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		response, err := db.SearchByPrefix(input, 10)
		if err != nil {
			os.Exit(exiterrrorREPLReading)
			return
		}
		fmt.Println(response)
		parsed, err := cql.ParseCQL("input", input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse")
			return
		}
		v, err := model.CreateEmbeddingNormalized(parsed.Normalize())
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR: ", err)
			os.Exit(exiterrrorREPLReading)
			return
		}

		abstract, err := db.FindSimilarQueries(v.Vector, 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to find: %s\n", err)
			os.Exit(exiterrrorREPLReading)
			return
		}
		for _, v := range abstract {
			fmt.Fprintf(os.Stderr, "%s: %01.2f\n", v.AbstractQuery, v.Score)
		}

	}
}

func runActionKlogImport(conf *cnf.Conf, srcPath string, fromDB bool, fromDate string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	targetDB, err := index.OpenDB(conf.IndexDataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open index: %s", err)
		os.Exit(exitErrorFailedToOpenIdex)
	}
	if fromDB {
		cp, err := dataimport.NewConcPersistence(conf.DataImportDB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open conc. persistence database: %s", err)
			os.Exit(exitErrorFailedToOpenQueryPersistence)
		}
		w2vModel, err := embedding.NewCQLEmbedder(conf.W2VModelPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open w2v model: %s", err)
			os.Exit(exitErrorFailedToOpenW2VModel)
		}

		if err := dataimport.ImportFromConcPersistence(
			ctx,
			cp,
			targetDB,
			conf.W2VSourceFilePath,
			w2vModel,
			fromDate,
		); err != nil {
			fmt.Fprintf(os.Stderr, "failed to import KonText log: %s", err)
			os.Exit(exitErrorImportFailed)
		}

	} else {
		if err := dataimport.ImportKontextLog(srcPath, targetDB); err != nil {
			fmt.Fprintf(os.Stderr, "failed to import KonText log: %s", err)
			os.Exit(exitErrorImportFailed)
		}
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
	importFromDB := cmdKlogImport.Bool("from-db", true, "if set, then the import will be performed from a configured SQL database (table kontext_conc_persistence)")
	importFromDate := cmdKlogImport.String("from-date", "", "if set, then cqlizer will read queries from a specified date (even if the index contains a previous import info)")
	cmdKlogImport.Usage = func() {
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
		conf := setup(cmdREPL.Arg(0))
		db, err := index.OpenDB(conf.IndexDataPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open index: %s", err)
			os.Exit(exitErrorFailedToOpenIdex)
		}
		model, err := embedding.NewCQLEmbedder(conf.W2VModelPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open model: %s", err)
			os.Exit(exitErrorFailedToOpenIdex)
		}
		runActionREPL(db, model)
	case actionKlogImport:
		cmdKlogImport.Parse(os.Args[2:])
		conf := setup(cmdKlogImport.Arg(0))

		runActionKlogImport(conf, cmdKlogImport.Arg(1), *importFromDB, *importFromDate)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action, please use 'help' to get more information")
	}

}
