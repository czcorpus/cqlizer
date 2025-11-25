// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/modutils"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

func ensureConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, ".config", "cqlizer")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return configDir, nil
}

func runActionREPL(modelType, modelPath string) {
	mlModel, err := eval.GetMLModel(modelType, modelPath)
	if err != nil {
		fmt.Printf("Error loading model: %v\n", err)
		os.Exit(1)
	}

	titleColor := color.New(color.FgHiMagenta).SprintFunc()
	greenColor := color.New(color.FgGreen).SprintFunc()
	redColor := color.New(color.FgRed).SprintFunc()

	// Default corpus size (can be overridden with 'set corpussize <value>')
	corpusSize := 6400000000.0 // 6.4G tokens default
	voteThreshold := 0.85
	lang := "cs"

	mlModel.SetClassThreshold(voteThreshold)

	fmt.Println("CQL Query Complexity Estimator")
	fmt.Println("Commands:")
	fmt.Println("  <CQL query>            - Estimate query execution time")
	fmt.Println("  set corpussize <size>  - Set corpus size (e.g., 'set corpussize 121826797')")
	fmt.Println("  set lang <lang>        - Set corpus language (e.g., 'set lang cs')")
	fmt.Println("  set vote <value 0..1>  - set model vote threshold")
	fmt.Println("  setup                  - view current settings")
	fmt.Println("  exit                   - Exit REPL")
	fmt.Printf("\nCurrent corpus size: %s tokens\n\n", modutils.FormatRoughSize(int64(corpusSize)))

	var historyFile string
	historyDir, err := ensureConfigDir()
	if err != nil {
		log.Error().Err(err).Msg("failed to determine user config directory - falling back to session-local history")

	} else {
		historyFile = filepath.Join(historyDir, "cql-history.txt")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      color.New(color.FgHiGreen).Sprintf("/cql> "),
		HistoryFile: historyFile,
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nCQLizer out!")
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}
		input := strings.TrimSpace(line)

		if input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if strings.HasPrefix(input, "set ") {
			parsedInput := strings.Fields(input)[1:]
			switch parsedInput[0] {
			case "corpussize":
				if len(parsedInput) == 2 {
					corpusSize, err = strconv.ParseFloat(parsedInput[1], 64)
					if err != nil {
						fmt.Println("Error: Invalid corpus size")
					}

				} else {
					fmt.Println("Usage: set corpussize <size>")
				}
			case "vote":
				if len(parsedInput) == 2 {
					voteThreshold, err = strconv.ParseFloat(parsedInput[1], 64)
					if err != nil {
						fmt.Println("failed to parse number")
					}
					mlModel.SetClassThreshold(voteThreshold)

				} else {
					fmt.Println("Usage: set vote <value 0..1>")
				}
			case "lang":
				if len(parsedInput) == 2 {
					lang = parsedInput[1]

				} else {
					fmt.Println("Usage: set lang <lang>")
				}
			default:
				fmt.Println("Unknown 'set' command")
			}
			continue

		} else if input == "setup" {
			fmt.Printf("%s:\t%s\n", titleColor("Corpus size"), modutils.FormatRoughSize(int64(corpusSize)))
			fmt.Printf("%s:\t\t%s\n", titleColor("Model"), modelPath)
			fmt.Printf("%s:\t%.2f\n", titleColor("Vote threshold"), voteThreshold)
			continue
		}

		// Treat as CQL query
		charProbs := feats.GetCharProbabilityProvider(lang)
		queryEval, err := feats.NewQueryEvaluation(input, corpusSize, 0, 0, charProbs)
		if err != nil {
			fmt.Printf("Error parsing CQL: %v\n", err)
			continue
		}

		// Display results

		fmt.Printf("%s:\n", titleColor("Pos. features"))
		for i, pos := range queryEval.Positions {
			fmt.Printf("  %s: wildcards=%0.2f, range=%d, smallCard=%d, numConcreteChars=%.2f, posNumAlts: %d\n",
				titleColor(fmt.Sprintf("[%d]", i)),
				pos.Regexp.WildcardScore, pos.Regexp.HasRange, pos.HasSmallCardAttr, pos.Regexp.NumConcreteChars, pos.NumAlternatives)
		}
		fmt.Printf("%s: glob=%d, meet=%d, union=%d, within=%d, containing=%d\n",
			titleColor("Global features"),
			queryEval.NumGlobConditions, queryEval.ContainsMeet,
			queryEval.ContainsUnion, queryEval.ContainsWithin, queryEval.ContainsContaining)

		if mlModel != nil {
			rfPRediction := mlModel.Predict(queryEval)
			var predResult string
			if rfPRediction.PredictedClass == 1 {
				predResult = redColor(rfPRediction.FastOrSlow() + " query")

			} else {
				predResult = greenColor(rfPRediction.FastOrSlow() + " query")
			}
			fmt.Printf("model prediction: %s\n", predResult)
			fmt.Printf("vote 0: %.2f, vote 1: %.2f\n", rfPRediction.Votes[0], rfPRediction.Votes[1])
		}
	}
}
