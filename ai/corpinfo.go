package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/czcorpus/rexplorer/parser"
)

type CorpInfoProvider struct {
	registryDirPath string
	regCache        map[string]*parser.Document
}

func (cp *CorpInfoProvider) GetRegistry(corpname string) (*parser.Document, error) {
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

func (cp *CorpInfoProvider) GetAttributes(corpname string) ([]string, error) {
	reg, err := cp.GetRegistry(corpname)
	if err != nil {
		return []string{}, err
	}
	ans := make([]string, len(reg.PosAttrs))
	for i, v := range reg.PosAttrs {
		ans[i] = v.Name
	}
	return ans, nil
}

func NewCorpInfoProvider(registryPath string) *CorpInfoProvider {
	return &CorpInfoProvider{
		registryDirPath: registryPath,
		regCache:        make(map[string]*parser.Document),
	}
}
