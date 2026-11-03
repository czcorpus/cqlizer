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

package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/czcorpus/cqlizer/dqd"
	"github.com/czcorpus/rexplorer/parser"
)

const lcpLayerInfoFileSuffix = ".json"

// corpusIDRegexp limits corpus IDs used as file names. The IDs come
// from a model, so they must not be able to point outside the configured
// directories.
var corpusIDRegexp = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func validateCorpusID(corpusID string) error {
	if !corpusIDRegexp.MatchString(corpusID) {
		return fmt.Errorf("invalid corpus ID %q", corpusID)
	}
	return nil
}

// CorpInfoProvider provides information about corpora of two kinds:
//   - Manatee corpora described by registry files (the default)
//   - LCP corpora described by layer info JSON files (<corpus>.json) in
//     a separate directory - a corpus is considered an LCP one if it has
//     such a file (see GetLCPLayerInfo)
type CorpInfoProvider struct {
	registryDirPath     string
	lcpLayerInfoDirPath string
	regCache            map[string]*parser.Document
}

func (cp *CorpInfoProvider) GetRegistry(corpname string) (*parser.Document, error) {
	if err := validateCorpusID(corpname); err != nil {
		return nil, err
	}
	curr, ok := cp.regCache[corpname]
	if ok {
		return curr, nil
	}
	data, err := os.ReadFile(filepath.Join(cp.registryDirPath, corpname))
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file for %s: %w", corpname, err)
	}
	doc, err := parser.ParseRegistryBytes(corpname, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry file for %s: %w", corpname, err)
	}
	return doc, nil
}

// GetLCPLayerInfo returns layer info of an LCP corpus. If the corpus is not
// an LCP one (or LCP corpora are not configured), isLCP is false.
func (cp *CorpInfoProvider) GetLCPLayerInfo(corpname string) (li *dqd.LayerInfo, isLCP bool, err error) {
	if cp.lcpLayerInfoDirPath == "" || validateCorpusID(corpname) != nil {
		return nil, false, nil
	}
	path := filepath.Join(cp.lcpLayerInfoDirPath, corpname+lcpLayerInfoFileSuffix)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	li, err = dqd.LoadLayerInfo(path)
	if err != nil {
		return nil, true, fmt.Errorf("failed to load layer info of LCP corpus %s: %w", corpname, err)
	}
	return li, true, nil
}

// NewCorpInfoProvider creates a provider of corpora information. An empty
// lcpLayerInfoPath means no LCP corpora are available.
func NewCorpInfoProvider(registryPath, lcpLayerInfoPath string) *CorpInfoProvider {
	return &CorpInfoProvider{
		registryDirPath:     registryPath,
		lcpLayerInfoDirPath: lcpLayerInfoPath,
		regCache:            make(map[string]*parser.Document),
	}
}
