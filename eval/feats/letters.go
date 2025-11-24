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

package feats

// ------------------------

type charProbabilityProvider interface {
	CharProbability(r rune) float64
}

// ------------------------

type charsProbabilityMap map[rune]float64

func (chmap charsProbabilityMap) CharProbability(r rune) float64 {
	v, ok := chmap[r]
	if ok {
		return v
	}
	return 1 / float64(len(chmap)) * 0.1
}

// --------

type fallbackCharProbProvider struct{}

func (fb fallbackCharProbProvider) CharProbability(r rune) float64 {
	return 1.0 / 30.0
}

// --------

// source: https://nlp.fi.muni.cz/cs/FrekvenceSlovLemmat

var czCharsProbs = charsProbabilityMap{
	'a': 6.698,
	'i': 4.571,
	's': 4.620,
	'á': 2.129,
	'í': 3.103,
	'š': 0.817,
	'b': 1.665,
	'j': 1.983,
	't': 5.554,
	'c': 1.601,
	'k': 3.752,
	'ť': 0.038,
	'č': 1.017,
	'l': 4.097,
	'u': 3.131,
	'd': 3.613,
	'm': 3.262,
	'ú': 0.145,
	'ď': 0.019,
	'n': 6.676,
	'ů': 0.569,
	'e': 7.831,
	'ň': 0.073,
	'v': 4.378,
	'é': 1.178,
	'o': 8.283,
	'w': 0.072,
	'ě': 1.491,
	'ó': 0.032,
	'x': 0.092,
	'f': 0.394,
	'p': 3.454,
	'y': 1.752,
	'g': 0.343,
	'q': 0.006,
	'ý': 0.942,
	'h': 1.296,
	'r': 3.977,
	'z': 2.123,
	'ř': 1.186,
	'ž': 1.022,
}

// source: https://en.wikipedia.org/wiki/Letter_frequency

var enCharsProbs = charsProbabilityMap{

	'a': 8.2,
	'b': 1.5,
	'c': 2.8,
	'd': 4.3,
	'e': 12.7,
	'f': 2.2,
	'g': 2.0,
	'h': 6.1,
	'i': 7.0,
	'j': 0.15,
	'k': 0.77,
	'l': 4.0,
	'm': 2.4,
	'n': 6.7,
	'o': 7.5,
	'p': 1.9,
	'q': 0.095,
	'r': 6.0,
	's': 6.3,
	't': 9.1,
	'u': 2.8,
	'v': 0.98,
	'w': 2.4,
	'x': 0.15,
	'y': 2.0,
	'z': 0.074,
}

// source: https://www.sttmedia.com/characterfrequency-german

var deCharsProbs = charsProbabilityMap{
	'a': 5.58,
	'ä': 0.54,
	'b': 1.96,
	'c': 3.16,
	'd': 4.98,
	'e': 16.93,
	'f': 1.49,
	'g': 3.02,
	'h': 4.98,
	'i': 8.02,
	'j': 0.24,
	'k': 1.32,
	'l': 3.60,
	'm': 2.55,
	'n': 10.53,
	'o': 2.24,
	'ö': 0.30,
	'p': 0.67,
	'q': 0.02,
	'r': 6.89,
	'ß': 0.37,
	's': 6.42,
	't': 5.79,
	'u': 3.83,
	'ü': 0.65,
	'v': 0.84,
	'w': 1.78,
	'x': 0.05,
	'y': 0.05,
	'z': 1.21,
}

// -----------------------

func GetCharProbabilityProvider(lang string) charProbabilityProvider {
	switch lang {
	case "cs":
		return czCharsProbs
	case "en":
		return enCharsProbs
	case "de":
		return deCharsProbs
	default:
		return fallbackCharProbProvider{}
	}
}
