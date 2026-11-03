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

package dqd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

// CorpusProps contains the corpus-schema-specific information CQL2DQD
// needs but that a CQL query itself never carries - DQD's Layer/attribute
// names are per-corpus (e.g. CQL "word" might be DQD "Token.form" in one
// corpus and "Token.text" in another), so they can't be derived from the
// query alone.
type CorpusProps struct {
	// PosAttrLayer is the DQD layer CQL positional attributes belong to
	// (almost always "Token").
	PosAttrLayer string `json:"posAttrLayer"`

	// SegmentLayer is the DQD layer all the tokens of a query are
	// part of (typically "Segment" - i.e. a sentence). It also serves
	// as the context of the "plain" result.
	SegmentLayer string `json:"segmentLayer"`

	// DefaultAttr is the CQL attribute a bare regexp position
	// (e.g. "dog" instead of [word="dog"]) applies to.
	DefaultAttr string `json:"defaultAttr"`

	// PosAttrs maps a CQL positional attribute name (e.g. "word") to its
	// DQD attribute name (e.g. "form").
	PosAttrs map[string]string `json:"posAttrs"`

	// Structures maps a CQL structure name (e.g. "text") to its DQD
	// layer name (e.g. "Text"). This can't be a simple capitalize() -
	// DQD layer names are corpus-defined, not derived from CQL names.
	Structures map[string]string `json:"structures"`

	// StructAttrs maps a CQL structure attribute name to its DQD
	// attribute name, keyed by CQL structure name.
	StructAttrs map[string]map[string]string `json:"structAttrs"`
}

// DefaultCorpusProps returns props matching a typical LCP corpus
// (UD-like token attributes, sentences as segments).
func DefaultCorpusProps() CorpusProps {
	return CorpusProps{
		PosAttrLayer: "Token",
		SegmentLayer: "Segment",
		DefaultAttr:  "word",
		PosAttrs: map[string]string{
			"word":  "form",
			"lemma": "lemma",
			"tag":   "xpos",
			"upos":  "upos",
		},
		Structures:  map[string]string{},
		StructAttrs: map[string]map[string]string{},
	}
}

// LoadCorpusProps loads props from a JSON file. Values found in the file
// are applied on top of DefaultCorpusProps - scalar values replace the
// defaults, map entries are added to (or replace entries of) the default
// maps.
func LoadCorpusProps(path string) (CorpusProps, error) {
	ans := DefaultCorpusProps()
	data, err := os.ReadFile(path)
	if err != nil {
		return ans, fmt.Errorf("failed to load corpus props: %w", err)
	}
	if err := json.Unmarshal(data, &ans); err != nil {
		return ans, fmt.Errorf("failed to load corpus props: %w", err)
	}
	return ans, nil
}

const (
	// cqlWordAttr is the CQL attribute for word forms. It is also the attribute
	// a bare regexp position ("dog") applies to.
	cqlWordAttr = "word"

	// dqdFormAttr is the usual name of the word form attribute in LCP corpora
	dqdFormAttr = "form"
)

// cqlComparableAttrTypes lists types of layer attributes which can be
// compared with a value using CQL's attr="value" (i.e. a string or a regexp).
// E.g. dictionaries (like "ufeat") or labels cannot be expressed this way.
var cqlComparableAttrTypes = []string{"text", "categorical"}

// CorpusPropsFromLayerInfo creates corpus props from LCP layer info.
//
// We expect a CQL query to use the real attribute names of the corpus
// (i.e. a query writer has examined the corpus info first), so the token
// attributes are mapped to themselves. The only exception is the CQL
// attribute "word" which is mapped to "form" (if the corpus has such an
// attribute and does not have its own "word").
//
// TODO: structures and their attributes are not filled in yet.
func CorpusPropsFromLayerInfo(li *LayerInfo) (CorpusProps, error) {
	tokenLayer, err := li.firstClassLayer("token", li.FirstClass.Token)
	if err != nil {
		return CorpusProps{}, err
	}
	if _, err := li.firstClassLayer("segment", li.FirstClass.Segment); err != nil {
		return CorpusProps{}, err
	}
	ans := CorpusProps{
		PosAttrLayer: li.FirstClass.Token,
		SegmentLayer: li.FirstClass.Segment,
		DefaultAttr:  cqlWordAttr,
		PosAttrs:     make(map[string]string),
		Structures:   map[string]string{},
		StructAttrs:  map[string]map[string]string{},
	}
	for name, attr := range tokenLayer.Attributes {
		if slices.Contains(cqlComparableAttrTypes, attr.Type) {
			ans.PosAttrs[name] = name
		}
	}
	if _, ok := ans.PosAttrs[cqlWordAttr]; !ok {
		if _, ok := ans.PosAttrs[dqdFormAttr]; ok {
			ans.PosAttrs[cqlWordAttr] = dqdFormAttr
		}
	}
	return ans, nil
}

// firstClassLayer returns a main layer (as specified in FirstClass)
// making sure it is really defined.
func (li *LayerInfo) firstClassLayer(role, name string) (*LayerDef, error) {
	if name == "" {
		return nil, fmt.Errorf("layer info does not specify the %s layer (firstClass.%s)", role, role)
	}
	layer, ok := li.Layers[name]
	if !ok {
		return nil, fmt.Errorf("the %s layer %s (firstClass.%s) is not defined in layer info", role, name, role)
	}
	return layer, nil
}
