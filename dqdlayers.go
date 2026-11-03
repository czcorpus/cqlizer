package main

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/czcorpus/cqlizer/dqd"
)

// runActionDQDLayers loads LCP layer info and prints a human-readable
// overview of what was understood from it.
func runActionDQDLayers(path string) {
	li, err := dqd.LoadLayerInfo(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	printLayerInfo(os.Stdout, li)
}

func printLayerInfo(w io.Writer, li *dqd.LayerInfo) {
	orMissing := func(v string) string {
		if v == "" {
			return "(not specified)"
		}
		return v
	}
	fmt.Fprintf(w, "main layers:\n")
	fmt.Fprintf(w, "    token:    %s\n", orMissing(li.FirstClass.Token))
	fmt.Fprintf(w, "    segment:  %s\n", orMissing(li.FirstClass.Segment))
	fmt.Fprintf(w, "    document: %s\n", orMissing(li.FirstClass.Document))

	fmt.Fprintf(w, "\nhierarchies:\n")
	for _, chain := range li.Hierarchies() {
		fmt.Fprintf(w, "    %s\n", strings.Join(chain, " > "))
	}

	fmt.Fprintf(w, "\nlayers:\n")
	for _, name := range li.LayerNames() {
		layer := li.Layers[name]
		fmt.Fprintf(w, "    %s (%s)\n", name, describeLayer(layer))
		for _, an := range layer.AttrNames() {
			fmt.Fprintf(w, "        %-24s %s\n", an, describeAttr(li, layer.Attributes[an]))
		}
		for _, an := range layer.MetaNames() {
			fmt.Fprintf(w, "        %-24s %s\n", "meta."+an, describeAttr(li, layer.Meta[an]))
		}
	}

	fmt.Fprintf(w, "\nCQL to DQD translation:\n")
	cp, err := dqd.CorpusPropsFromLayerInfo(li)
	if err != nil {
		fmt.Fprintf(w, "    not available: %s\n", err)

	} else {
		fmt.Fprintf(w, "    token layer:   %s\n", cp.PosAttrLayer)
		fmt.Fprintf(w, "    segment layer: %s\n", cp.SegmentLayer)
		fmt.Fprintf(w, "    CQL attributes:\n")
		for _, name := range slices.Sorted(maps.Keys(cp.PosAttrs)) {
			if cp.PosAttrs[name] == name {
				fmt.Fprintf(w, "        %s\n", name)

			} else {
				fmt.Fprintf(w, "        %s (-> %s)\n", name, cp.PosAttrs[name])
			}
		}
		fmt.Fprintf(w, "    CQL structures:\n")
		for _, name := range slices.Sorted(maps.Keys(cp.Structures)) {
			sp := cp.Structures[name]
			var position string
			switch {
			case sp.Layer == cp.SegmentLayer:
				position = "the segment"
			case sp.ContainsSegments:
				position = "contains segments"
			default:
				position = "inside segments"
			}
			fmt.Fprintf(w, "        <%s> (%s): %s\n", name, position, strings.Join(slices.Sorted(maps.Keys(sp.Attrs)), ", "))
		}
	}

	if len(li.GlobalAttrs) > 0 {
		fmt.Fprintf(w, "\nglobal attributes:\n")
		for _, name := range slices.Sorted(maps.Keys(li.GlobalAttrs)) {
			gattr := li.GlobalAttrs[name]
			fmt.Fprintf(w, "    %s (%s): %d keys\n", name, gattr.Type, len(gattr.Keys))
		}
	}
}

func describeLayer(layer *dqd.LayerDef) string {
	var props []string
	if layer.LayerType != "" {
		props = append(props, string(layer.LayerType))
	}
	if layer.Contains != "" {
		props = append(props, "contains "+layer.Contains)
	}
	if layer.PartOf != "" {
		props = append(props, "part of "+layer.PartOf)
	}
	if layer.Anchoring != nil {
		var anchors []string
		if layer.Anchoring.Stream {
			anchors = append(anchors, "stream")
		}
		if layer.Anchoring.Time {
			anchors = append(anchors, "time")
		}
		if layer.Anchoring.Location {
			anchors = append(anchors, "location")
		}
		props = append(props, "anchored to "+strings.Join(anchors, "+"))
	}
	return strings.Join(props, ", ")
}

func describeAttr(li *dqd.LayerInfo, attr *dqd.LayerAttr) string {
	switch {
	case attr.IsRef():
		return "-> global attribute " + attr.Ref
	case attr.IsRelationEndpoint():
		return fmt.Sprintf("-> %s (%s)", attr.Entity, attr.DisplayName)
	}
	ans := attr.Type
	switch {
	case len(attr.Values) > 0:
		ans += fmt.Sprintf(", %d values", len(attr.Values))
	case attr.IsGlobal && len(li.GlobalAttrValues[attr.Name]) > 0:
		ans += fmt.Sprintf(", %d values (global)", len(li.GlobalAttrValues[attr.Name]))
	case attr.IsGlobal:
		ans += ", global"
	}
	if attr.DisplayName != "" {
		ans += fmt.Sprintf(" [%s]", attr.DisplayName)
	}
	if attr.Nullable {
		ans += ", nullable"
	}
	return ans
}
