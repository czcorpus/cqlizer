// Copyright 2026 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2026 Department of Linguistics,
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

// udgen reduces the UD validator's language-specific data files
// (https://github.com/UniversalDependencies/tools/tree/master/data)
// into the compact ud_data.json embedded by the tagsets package.
//
// Usage:
//
//	curl -sSfLO https://raw.githubusercontent.com/UniversalDependencies/tools/master/data/feats.json
//	curl -sSfLO https://raw.githubusercontent.com/UniversalDependencies/tools/master/data/deprels.json
//	go run ./tagsets/udgen feats.json deprels.json > tagsets/data/ud_data.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

type featSpec struct {
	Permitted int      `json:"permitted"`
	UValues   []string `json:"uvalues"`
	LValues   []string `json:"lvalues"`
}

type deprelSpec struct {
	Permitted int `json:"permitted"`
}

type output struct {
	Source    string                         `json:"source"`
	Retrieved string                         `json:"retrieved"`
	Feats     map[string]map[string][]string `json:"feats"`
	Deprels   map[string][]string            `json:"deprels"`
}

func loadJSON(path string, v any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.Unmarshal(data, v); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse %s: %s\n", path, err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: udgen feats.json deprels.json")
		os.Exit(1)
	}
	var feats struct {
		Features map[string]map[string]featSpec `json:"features"`
	}
	loadJSON(os.Args[1], &feats)
	var deprels struct {
		Deprels map[string]map[string]deprelSpec `json:"deprels"`
	}
	loadJSON(os.Args[2], &deprels)

	out := output{
		Source:    "https://github.com/UniversalDependencies/tools/tree/master/data",
		Retrieved: time.Now().Format("2006-01-02"),
		Feats:     make(map[string]map[string][]string),
		Deprels:   make(map[string][]string),
	}
	for lang, fs := range feats.Features {
		m := make(map[string][]string)
		for name, spec := range fs {
			if spec.Permitted == 0 {
				continue
			}
			vals := append(append([]string{}, spec.UValues...), spec.LValues...)
			sort.Strings(vals)
			m[name] = vals
		}
		if len(m) > 0 {
			out.Feats[lang] = m
		}
	}
	for lang, ds := range deprels.Deprels {
		var lst []string
		for name, spec := range ds {
			if spec.Permitted != 0 {
				lst = append(lst, name)
			}
		}
		if len(lst) > 0 {
			sort.Strings(lst)
			out.Deprels[lang] = lst
		}
	}
	// encoding/json sorts map keys, so the output is stable
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
