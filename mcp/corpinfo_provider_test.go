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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/czcorpus/rexplorer/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRegistry = `NAME "Test corpus"
ATTRIBUTE word
ATTRIBUTE lemma
`

func writeRegistry(t *testing.T, dir, corpusID string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, corpusID), []byte(testRegistry), 0o644))
}

func TestGetRegistryInvalidCorpusID(t *testing.T) {
	prov := NewCorpInfoProvider(t.TempDir())
	for _, corpusID := range []string{"", "../etc", "a/b", ".hidden"} {
		_, err := prov.GetRegistry(corpusID)
		assert.ErrorContains(t, err, "invalid corpus ID", corpusID)
	}
}

func TestGetRegistryCache(t *testing.T) {
	dir := t.TempDir()
	prov := NewCorpInfoProvider(dir)

	// a failure is not cached - the registry may be added later
	_, err := prov.GetRegistry("test")
	assert.ErrorContains(t, err, "failed to read registry file")
	writeRegistry(t, dir, "test")

	reg, err := prov.GetRegistry("test")
	require.NoError(t, err)
	assert.Equal(t, "Test corpus", string(reg.GetProperty("NAME").Value()))

	// a parsed registry is cached - no more file reads
	require.NoError(t, os.Remove(filepath.Join(dir, "test")))
	cached, err := prov.GetRegistry("test")
	require.NoError(t, err)
	assert.Same(t, reg, cached)
}

func TestGetRegistryConcurrent(t *testing.T) {
	dir := t.TempDir()
	writeRegistry(t, dir, "test")
	prov := NewCorpInfoProvider(dir)

	const numCalls = 20
	regs := make([]*parser.Document, numCalls)
	var wg sync.WaitGroup
	for i := range numCalls {
		wg.Go(func() {
			reg, err := prov.GetRegistry("test")
			assert.NoError(t, err)
			regs[i] = reg
		})
	}
	wg.Wait()
	// concurrent first requests all get the single cached instance
	for _, reg := range regs {
		assert.Same(t, regs[0], reg)
	}
}
