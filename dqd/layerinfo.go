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
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
)

// LayerInfo describes the layers (i.e. the structure) of an LCP corpus
// as produced by LCP ("layer info" / corpus config JSON).
//
// The format is not formally specified and the available samples differ
// in which fields they contain, so the model is lenient - unknown fields
// are ignored and all the fields are optional.
type LayerInfo struct {
	Layers map[string]*LayerDef `json:"layer"`

	// FirstClass names the main layers of the corpus
	FirstClass FirstClass `json:"firstClass"`

	// GlobalAttrValues lists values of global categorical attributes
	// (i.e. attributes with LayerAttr.IsGlobal == true, e.g. "upos"
	// or "udep").
	GlobalAttrValues map[string][]string `json:"glob_attr"`

	// GlobalAttrs defines global attributes referenced by layers
	// via LayerAttr.Ref (e.g. a speaker described by a dictionary).
	GlobalAttrs map[string]*GlobalAttrDef `json:"globalAttributes"`
}

// FirstClass names the main layers of a corpus. The values are layer
// names (keys of LayerInfo.Layers), e.g. a corpus of spoken language may
// use "Utterance" as its segment layer.
type FirstClass struct {
	Token    string `json:"token"`
	Segment  string `json:"segment"`
	Document string `json:"document"`
}

type LayerType string

const (
	// LayerTypeUnit is a layer of atomic units (e.g. Token)
	LayerTypeUnit LayerType = "unit"

	// LayerTypeSpan is a layer containing other layers (e.g. Segment)
	LayerTypeSpan LayerType = "span"

	// LayerTypeRelation is a relation between units (e.g. DepRel)
	LayerTypeRelation LayerType = "relation"
)

// LayerDef describes a single layer.
//
// Note: the "alignment" property (used by parallel corpora) is not modeled
// as its form varies (a boolean, a string or an object).
type LayerDef struct {
	// Name is the layer name (the key in LayerInfo.Layers)
	Name string `json:"-"`

	// LayerType is empty for language-specific layers (see PartOf)
	LayerType LayerType `json:"layerType"`

	// Contains names the layer directly below this one (e.g. Segment
	// contains Token)
	Contains string `json:"contains"`

	// PartOf is used by language-specific variants of a layer in parallel
	// corpora (e.g. layer "Token@de" is part of "Token")
	PartOf string `json:"partOf"`

	Abstract bool `json:"abstract"`

	Anchoring *Anchoring `json:"anchoring"`

	// Attributes of the layer (with the exception of the "meta" group)
	Attributes map[string]*LayerAttr `json:"-"`

	// Meta attributes are listed in the layer info as a separate group
	// ("meta"), but DQD queries refer to them just by their name (e.g.
	// "classCode = /^S/" for Document's meta.classCode).
	Meta map[string]*LayerAttr `json:"-"`
}

// metaAttrsKey is the key of the group of meta attributes within
// the "attributes" of a layer
const metaAttrsKey = "meta"

// UnmarshalJSON splits layer attributes into regular ones and meta ones.
func (ld *LayerDef) UnmarshalJSON(data []byte) error {
	type plainLayerDef LayerDef // prevents recursion to this method
	var tmp struct {
		plainLayerDef
		Attributes map[string]json.RawMessage `json:"attributes"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*ld = LayerDef(tmp.plainLayerDef)
	for name, raw := range tmp.Attributes {
		if name == metaAttrsKey {
			if err := json.Unmarshal(raw, &ld.Meta); err != nil {
				return fmt.Errorf("invalid meta attributes: %w", err)
			}
			continue
		}
		var attr LayerAttr
		if err := json.Unmarshal(raw, &attr); err != nil {
			return fmt.Errorf("invalid attribute %s: %w", name, err)
		}
		if ld.Attributes == nil {
			ld.Attributes = make(map[string]*LayerAttr)
		}
		ld.Attributes[name] = &attr
	}
	for name, attr := range ld.Attributes {
		attr.Name = name
	}
	for name, attr := range ld.Meta {
		attr.Name = name
	}
	return nil
}

// AttrNames returns sorted names of the layer's attributes
// (without the meta ones).
func (ld *LayerDef) AttrNames() []string {
	return slices.Sorted(maps.Keys(ld.Attributes))
}

// MetaNames returns sorted names of the layer's meta attributes.
func (ld *LayerDef) MetaNames() []string {
	return slices.Sorted(maps.Keys(ld.Meta))
}

// Anchoring tells how a layer is anchored - to the token stream,
// to time (e.g. audio/video) and/or to a location (e.g. an image).
type Anchoring struct {
	Stream   bool `json:"stream"`
	Time     bool `json:"time"`
	Location bool `json:"location"`
}

// LayerAttr describes a layer attribute. Besides regular (typed)
// attributes, there are two special kinds - a reference to a global
// attribute (see IsRef) and an endpoint of a relation (see
// IsRelationEndpoint).
type LayerAttr struct {
	// Name is the attribute name (the key within the layer attributes)
	Name string `json:"-"`

	// Type is e.g. "text", "categorical", "dict", "labels", "integer",
	// "number", "boolean", "date". It is empty for global attribute
	// references and relation endpoints.
	Type string `json:"type"`

	// DisplayName is a human-readable name (e.g. a tagset name
	// "Penn Treebank", or "head"/"dependent" for relation endpoints)
	DisplayName string `json:"name"`

	// Values lists possible values of a categorical attribute. For global
	// attributes, values are listed in LayerInfo.GlobalAttrValues instead.
	Values []string `json:"values"`

	IsGlobal   bool `json:"isGlobal"`
	Nullable   bool `json:"nullable"`
	IsOptional bool `json:"isOptional"`

	// NLabels is the number of labels of a "labels" attribute
	NLabels int `json:"nlabels"`

	// Ref names a global attribute (see LayerInfo.GlobalAttrs)
	Ref string `json:"ref"`

	// Entity names the layer a relation endpoint refers to
	Entity string `json:"entity"`
}

// IsRef tells whether the attribute is a reference to a global attribute.
func (la *LayerAttr) IsRef() bool {
	return la.Ref != ""
}

// IsRelationEndpoint tells whether the attribute is an endpoint (source
// or target) of a relation layer.
func (la *LayerAttr) IsRelationEndpoint() bool {
	return la.Entity != ""
}

// GlobalAttrDef describes a global attribute - typically a dictionary
// (Type == "dict") with typed keys.
type GlobalAttrDef struct {
	Type string                `json:"type"`
	Keys map[string]*LayerAttr `json:"keys"`
}

// ParseLayerInfo parses LCP layer info JSON.
func ParseLayerInfo(data []byte) (*LayerInfo, error) {
	var ans LayerInfo
	if err := json.Unmarshal(data, &ans); err != nil {
		return nil, fmt.Errorf("failed to parse layer info: %w", err)
	}
	if len(ans.Layers) == 0 {
		return nil, errors.New("failed to parse layer info: no layers found")
	}
	for name, layer := range ans.Layers {
		if layer == nil {
			return nil, fmt.Errorf("failed to parse layer info: empty definition of layer %s", name)
		}
		layer.Name = name
	}
	for _, gattr := range ans.GlobalAttrs {
		for name, key := range gattr.Keys {
			key.Name = name
		}
	}
	return &ans, nil
}

// LoadLayerInfo loads LCP layer info from a JSON file.
func LoadLayerInfo(path string) (*LayerInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load layer info: %w", err)
	}
	return ParseLayerInfo(data)
}

// LayerNames returns sorted names of all the layers.
func (li *LayerInfo) LayerNames() []string {
	return slices.Sorted(maps.Keys(li.Layers))
}

// Hierarchies returns chains of layers connected by the "contains"
// relation, each starting with a layer not contained in any other layer
// (e.g. [Document Division Paragraph Segment Token]). Layers not involved
// in any containment (e.g. relations) are not part of any chain.
func (li *LayerInfo) Hierarchies() [][]string {
	contained := make(map[string]bool)
	for _, layer := range li.Layers {
		if layer.Contains != "" {
			contained[layer.Contains] = true
		}
	}
	var ans [][]string
	for _, name := range li.LayerNames() {
		if contained[name] || li.Layers[name].Contains == "" {
			continue
		}
		chain := []string{name}
		for curr := li.Layers[name]; curr != nil && curr.Contains != ""; curr = li.Layers[curr.Contains] {
			if slices.Contains(chain, curr.Contains) {
				break // a cycle - should not happen in valid data
			}
			chain = append(chain, curr.Contains)
		}
		ans = append(ans, chain)
	}
	return ans
}
