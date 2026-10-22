// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Department of Linguistics,
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

package cql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// firstAtomQuery digs out the AtomQuery of the first position in a
// single-sequence query, e.g. the one matched by `[word="x"]`.
func firstAtomQuery(t *testing.T, q *Query) *AtomQuery {
	t.Helper()
	aq := q.Sequence.Seq[0].Repetition[0].Variant1.AtomQuery
	assert.NotNil(t, aq)
	return aq
}

func nodesOfType[T ASTNode](q *Query) []T {
	var ans []T
	q.ForEachElement(func(parent, v ASTNode) {
		if tv, ok := v.(T); ok {
			ans = append(ans, tv)
		}
	})
	return ans
}

func TestPositionAccessors(t *testing.T) {
	q, err := ParseCQL("test", `[word="a"]`)
	assert.NoError(t, err)
	pos := firstAtomQuery(t, q).Position()
	assert.NotNil(t, pos.OnePosition())
	assert.Nil(t, pos.NumberedPosition())

	q, err = ParseCQL("test", `(union 1:[word="a"] 2:[word="b"])`)
	assert.NoError(t, err)
	mu := firstAtomQuery(t, q).Position().OnePosition().Variant5.MuPart
	assert.NotNil(t, mu.Variant1)
	np1 := mu.Variant1.UnionOp.Position1
	assert.Nil(t, np1.OnePosition())
	assert.NotNil(t, np1.NumberedPosition())
	assert.Equal(t, ASTString("1"), np1.NumberedPosition().Number)
}

func TestAtomQueryAccessors(t *testing.T) {
	q, err := ParseCQL("test", `[word="a"]`)
	assert.NoError(t, err)
	aq := firstAtomQuery(t, q)
	assert.NotNil(t, aq.Position())
	assert.Nil(t, aq.Sequence())
	assert.Nil(t, aq.WithinContainingBlocks())

	q, err = ParseCQL("test", `([word="a"] within <s>)`)
	assert.NoError(t, err)
	aq = firstAtomQuery(t, q)
	assert.Nil(t, aq.Position())
	assert.NotNil(t, aq.Sequence())
	assert.Equal(t, `[word="a"]`, aq.Sequence().CQL())
	blocks := aq.WithinContainingBlocks()
	assert.Len(t, blocks, 1)
	assert.False(t, blocks[0].Not)
	assert.Equal(t, ASTString("within"), blocks[0].Keyword)
	assert.Equal(t, `<s>`, blocks[0].WithinContainingPart.CQL())
}

func TestGlobCondAccessors(t *testing.T) {
	q, err := ParseCQL("test", `[word="a"] & 1.word=2.word`)
	assert.NoError(t, err)
	gc := q.GlobPart.GlobCond[0]
	attr, ok := gc.AttrCompare()
	assert.True(t, ok)
	assert.Equal(t, GlobCondAttrCompare{
		Number1: "1", AttName1: "word", Not: false, Op: "=",
		Number2: "2", AttName2: "word",
	}, attr)
	_, ok = gc.FreqCompare()
	assert.False(t, ok)

	q, err = ParseCQL("test", `[word="a"] & f(1.word)>=5`)
	assert.NoError(t, err)
	gc = q.GlobPart.GlobCond[0]
	freq, ok := gc.FreqCompare()
	assert.True(t, ok)
	assert.Equal(t, GlobCondFreqCompare{
		Number: "1", AttName: "word", Not: false, Op: ">=", Threshold: "5",
	}, freq)
	_, ok = gc.AttrCompare()
	assert.False(t, ok)
}

func TestWithinContainingPartAccessors(t *testing.T) {
	q, err := ParseCQL("test", `[word="a"] within [word="b"]`)
	assert.NoError(t, err)
	wcp := q.WithinOrContaining[0].WithinContainingPart
	assert.NotNil(t, wcp.Sequence())
	assert.Nil(t, wcp.WithinNumber())
	assert.Nil(t, wcp.AlignedPart())
	assert.Equal(t, `[word="b"]`, wcp.Sequence().CQL())

	q, err = ParseCQL("test", `[word="a"] within 5`)
	assert.NoError(t, err)
	wcp = q.WithinOrContaining[0].WithinContainingPart
	assert.Nil(t, wcp.Sequence())
	assert.NotNil(t, wcp.WithinNumber())
	assert.Nil(t, wcp.AlignedPart())
	assert.Equal(t, ASTString("5"), wcp.WithinNumber().Value)

	q, err = ParseCQL("test", `[word="car"] within europarl5_de: [word="Auto"]`)
	assert.NoError(t, err)
	wcp = q.WithinOrContaining[0].WithinContainingPart
	assert.Nil(t, wcp.Sequence())
	assert.Nil(t, wcp.WithinNumber())
	assert.NotNil(t, wcp.AlignedPart())
	assert.Equal(t, ASTString("europarl5_de"), wcp.AlignedPart().AttName)
	assert.Equal(t, `[word="Auto"]`, wcp.AlignedPart().Sequence.CQL())
}

func TestRgCharAccessors(t *testing.T) {
	q, err := ParseCQL("test", `"ab.*c?^"`)
	assert.NoError(t, err)
	chars := nodesOfType[*RgChar](q)
	assert.Len(t, chars, 7)

	lit, ok := chars[0].Literal()
	assert.True(t, ok)
	assert.Equal(t, ASTString("a"), lit)
	lit, ok = chars[1].Literal()
	assert.True(t, ok)
	assert.Equal(t, ASTString("b"), lit)

	assert.NotNil(t, chars[2].Any())
	assert.NotNil(t, chars[3].Repeat())

	lit, ok = chars[4].Literal()
	assert.True(t, ok)
	assert.Equal(t, ASTString("c"), lit)

	assert.NotNil(t, chars[5].QM())
	op, ok := chars[6].Op()
	assert.True(t, ok)
	assert.Equal(t, ASTString("^"), op)

	// non-matching accessors return nil/false on the "wrong" variant
	_, ok = chars[2].Literal()
	assert.False(t, ok)
	_, ok = chars[0].Op()
	assert.False(t, ok)
}

func TestRgAltValAccessors(t *testing.T) {
	q, err := ParseCQL("test", `"[abc]"`)
	assert.NoError(t, err)
	alts := nodesOfType[*RgAltVal](q)
	assert.Len(t, alts, 3)
	for i, want := range []string{"a", "b", "c"} {
		char := alts[i].Char()
		assert.NotNil(t, char)
		lit, ok := char.Literal()
		assert.True(t, ok)
		assert.Equal(t, ASTString(want), lit)
	}

	q, err = ParseCQL("test", `"[a-z]"`)
	assert.NoError(t, err)
	alts = nodesOfType[*RgAltVal](q)
	assert.Len(t, alts, 1)
	from, to, ok := alts[0].Range()
	assert.True(t, ok)
	assert.Equal(t, ASTString("a"), from)
	assert.Equal(t, ASTString("z"), to)

	// a leading "-" can't be part of a range (no preceding AnyLetter), so
	// it takes the dash-literal alternative (RgAltVal variant2).
	q, err = ParseCQL("test", `"[-az]"`)
	assert.NoError(t, err)
	alts = nodesOfType[*RgAltVal](q)
	assert.Len(t, alts, 3)
	lit, ok := alts[0].Literal()
	assert.True(t, ok)
	assert.Equal(t, ASTString("-"), lit)
	assert.Nil(t, alts[0].Char())
	_, _, ok = alts[0].Range()
	assert.False(t, ok)
}

// TestRgOpVsAnchor pins down the RG_ANCHOR/RG_OP split: only "^" and "$"
// are true regexp operators/anchors (RgChar.Op()); dash, comma,
// underscore and digits are plain literal characters
// (RgChar.Literal()), both inside and outside character classes.
func TestRgOpVsAnchor(t *testing.T) {
	q, err := ParseCQL("test", `"a-b,c_d1^e$f"`)
	assert.NoError(t, err)
	chars := nodesOfType[*RgChar](q)

	wantLiterals := map[int]string{0: "a", 1: "-", 2: "b", 3: ",", 4: "c", 5: "_", 6: "d", 7: "1", 9: "e", 11: "f"}
	for i, want := range wantLiterals {
		lit, ok := chars[i].Literal()
		assert.True(t, ok, "index %d", i)
		assert.Equal(t, ASTString(want), lit)
		assert.True(t, chars[i].IsConstant())
	}

	for _, i := range []int{8, 10} {
		_, ok := chars[i].Op()
		assert.True(t, ok, "index %d", i)
		assert.False(t, chars[i].IsConstant())
	}
	op, ok := chars[8].Op()
	assert.True(t, ok)
	assert.Equal(t, ASTString("^"), op)
	op, ok = chars[10].Op()
	assert.True(t, ok)
	assert.Equal(t, ASTString("$"), op)
}
