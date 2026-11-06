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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/czcorpus/rexplorer/parser"
)

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

// CorpInfoProvider provides information about Manatee corpora described
// by registry files. LCP corpora are unknown to it - the DQD tools get
// their layer infos from the caller (see addDQDTools).
//
// Parsed registries are cached for the provider's lifetime - a changed
// registry file takes effect after a restart. The provider is safe for
// concurrent use (MCP tool handlers may run concurrently).
type CorpInfoProvider struct {
	registryDirPath string
	regCacheMu      sync.RWMutex
	// regCache contains parsed registries shared by all callers, so they
	// must be treated as read-only
	regCache map[string]*parser.Document
}

// GetRegistry returns the parsed registry of a corpus. Failures are not
// cached - a missing registry file may be added later.
func (cp *CorpInfoProvider) GetRegistry(corpname string) (*parser.Document, error) {
	if err := validateCorpusID(corpname); err != nil {
		return nil, err
	}
	cp.regCacheMu.RLock()
	curr, ok := cp.regCache[corpname]
	cp.regCacheMu.RUnlock()
	if ok {
		return curr, nil
	}
	// Loading outside the lock means concurrent first requests for the same
	// corpus may parse its registry more than once. That is harmless (the
	// results are equal) and keeps slow file reads from blocking other corpora.
	data, err := os.ReadFile(filepath.Join(cp.registryDirPath, corpname))
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file for %s: %w", corpname, err)
	}
	doc, err := parser.ParseRegistryBytes(corpname, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry file for %s: %w", corpname, err)
	}
	cp.regCacheMu.Lock()
	defer cp.regCacheMu.Unlock()
	if curr, ok := cp.regCache[corpname]; ok {
		// another request was faster - keep a single shared instance
		return curr, nil
	}
	cp.regCache[corpname] = doc
	return doc, nil
}

// NewCorpInfoProvider creates a provider of corpora information.
func NewCorpInfoProvider(registryPath string) *CorpInfoProvider {
	return &CorpInfoProvider{
		registryDirPath: registryPath,
		regCache:        make(map[string]*parser.Document),
	}
}
