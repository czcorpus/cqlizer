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

	// Structures maps a CQL structure name (e.g. "doc") to its DQD
	// properties.
	Structures map[string]StructProps `json:"structures"`
}

// StructProps describes how a CQL structure maps to DQD.
type StructProps struct {
	// Layer is the DQD layer of the structure (e.g. "Document"). It can't
	// be derived from the CQL name - DQD layer names are corpus-defined.
	Layer string `json:"layer"`

	// ContainsSegments tells whether the structure is above the segment
	// layer (e.g. a document containing sentences) - in such case, the
	// segment is part of the structure. Otherwise (e.g. a named entity
	// inside a sentence), the structure is part of the segment and the
	// tokens are part of the structure. A structure with the same layer
	// as CorpusProps.SegmentLayer is the segment itself.
	ContainsSegments bool `json:"containsSegments"`

	// Attrs maps CQL structural attribute names to DQD attribute names.
	Attrs map[string]string `json:"attrs"`
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
		Structures: map[string]StructProps{
			"s":   {Layer: "Segment"},
			"doc": {Layer: "Document", ContainsSegments: true},
		},
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
var cqlComparableAttrTypes = []string{"text", "categorical", "string"}

// CorpusPropsFromLayerInfo creates corpus props from LCP layer info.
//
// We expect a CQL query to use the real attribute names of the corpus
// (i.e. a query writer has examined the corpus info first), so the token
// attributes are mapped to themselves. The only exception is the CQL
// attribute "word" which is mapped to "form" (if the corpus has such an
// attribute and does not have its own "word").
//
// For the same reason, span layers become CQL structures of the same
// name and their attributes (including the meta ones which DQD refers to
// just by their names) are mapped to themselves.
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
		Structures:   make(map[string]StructProps),
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
	for name, layer := range li.Layers {
		if layer.LayerType != LayerTypeSpan {
			continue
		}
		sp := StructProps{
			Layer:            name,
			ContainsSegments: li.containsLayer(name, ans.SegmentLayer),
			Attrs:            make(map[string]string),
		}
		for _, attrs := range []map[string]*LayerAttr{layer.Attributes, layer.Meta} {
			for aname, attr := range attrs {
				if slices.Contains(cqlComparableAttrTypes, attr.Type) {
					sp.Attrs[aname] = aname
				}
			}
		}
		ans.Structures[name] = sp
	}
	return ans, nil
}

// containsLayer tells whether layer outer contains (directly or
// indirectly) layer inner.
func (li *LayerInfo) containsLayer(outer, inner string) bool {
	visited := map[string]bool{outer: true}
	for curr := li.Layers[outer]; curr != nil && curr.Contains != ""; curr = li.Layers[curr.Contains] {
		if curr.Contains == inner {
			return true
		}
		if visited[curr.Contains] {
			return false // a cycle - should not happen in valid data
		}
		visited[curr.Contains] = true
	}
	return false
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
