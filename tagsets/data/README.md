# Tagset data

All the files are embedded into the binary.

* `ud_data.json` - permitted UD features, their values and dependency relations
  for each language, reduced from the
  [UD validator data](https://github.com/UniversalDependencies/tools/tree/master/data)
  by `../udgen` (see its header for instructions).
* `kontext_tagsets.xml` - descriptions of Czech positional tagsets in the format
  used by KonText (labels of positions and values).
* `cqlizer_tagsets.xml` - descriptions of tagsets missing in KonText (`verbtag`,
  based on the Alpha project; `cs_cnc2020_spk`, i.e. cs_cnc2020 extended with
  spoken parts of speech used in ORTOFON v3), the same format plus an `extends`
  attribute for inheriting descriptions of another tagset.
* `positional/{tagset}/{corpus}` - all the distinct tags found in large reference
  corpora (one tag per line, multiple tags of a token may be joined by `|`).
  The directory name is the tagset identifier (the same as in KonText) usable
  in the CQLizer configuration (`corporaProps.{corpus}.tagsets`). To add a tagset
  or improve coverage, just add a file; files of a single tagset are merged.
  The number of positions is determined from the data (e.g. the XML declares
  `cs_cnc2000_spk` as 16-positional while the actual tags have 15 positions).
  The `verbtag` values come from `syn_r25` (recent tagging) and `syn2020`
  (which additionally uses `--p---` for deverbal adjectives).
