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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/rf"
)

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
