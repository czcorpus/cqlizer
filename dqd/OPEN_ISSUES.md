# CQL to DQD translation - open issues

Status as of 2026-11-03 (steps 0-7 of the original plan are done: renderer,
values, boolean logic, sequences, repetition, within/containing, corpus
props from LCP layer info).

## Not verified on a real LCP instance yet

The DQD grammar check (`scripts/dqd_check.py`) only verifies syntax. LCP also
validates queries on the server side (e.g. which labels can be referenced),
so these still need to be tried:

- **Two `@` relations on one unit** - `Token@s@namedEntity1` (tokens within
  a structure *inside* the segment, e.g. a named entity). Allowed by the
  grammar, but not used in any LiRI sample.
- **A structure label as a result entity** - `entities document1` for
  `<Document .../> containing ...`. LiRI samples only use the segment label
  (`entities s`).
- **`!EXISTS` around a token part** - `<Segment/> !containing [...]`. Mirrors
  the `NOT EXISTS` sample (row 12 of the LiRI battery), but not tried with
  our output.
- **Direction of `@` for layers containing segments** - layer info of one
  LiRI corpus says `Topic contains Segment`, but its sample query writes
  `Topic@s top` (i.e. the opposite of what we produce: `Segment@topic1 s`).
  Possibly just a sloppy sample.

## Pending design decisions

- **Highlighting of the whole match** - the plain result lists tokens outside
  repetitions only (LCP forbids referencing labels inside repeated sequences),
  or `*` if there are none. A LiRI sample uses a top-level sequence label as
  the entity (`entities seq` with `sequence 0..*` inside) - this could
  highlight the whole match like KonText does. Needs a test on LCP.
- **Matches across sentence boundaries** - all tokens are part of the segment
  (`@s`), so unlike CQL, a match cannot cross a segment boundary. Could become
  a CorpusProps option.
- **`DefaultCorpusProps()` maps `tag` -> `xpos`** - a name-only mapping which
  does not work in general (different tagsets). It is used only when no layer
  info/props are provided. Remove?
- **Invalid IR panics in `Query.String()`** - could return an error instead
  (`Render() (string, error)`).
- **`!within`** - "the segment is not part of any such structure" would need
  a reference from inside a quantification to the outer segment in the
  opposite direction; not sure DQD can express it.
- **DQD grammar file** - `dqd/dqd_grammar.lark` cannot be part of the project
  (license); a helper script downloading a pinned version from LCP's git is
  planned.

## Unsupported CQL features

Each produces an `UnsupportedFeature` error with a specific message.

- global conditions and numbered positions (`1:[...] & 1.tag = 2.tag`) -
  planned (step 8 of the original plan); `f(...)` frequency conditions have no
  DQD counterpart (frequency exists only in analysis results)
- alternative sequences (`seq1 | seq2`), negated sequences
- structure tags inside a sequence (`<s> [...]`, `[...] </s>`) - could become
  `start()`/`end()` comparisons
- meet/union, `~` positions, `ws()`, `term()`, `swap()`, `ccoll()`, `#n`
- operators `<=`, `>=` in attribute conditions
- regexps with unicode properties (`\p{...}`) or non-leading inline flags
- `within`: negated, a token query, a number of tokens, aligned (parallel)
  parts, inside a parenthesized sub-query
- `containing`: after a token query, a structure, a structure alone as a query
- token/structure attributes other than text/categorical/string (e.g. `ufeat`
  dictionaries, labels, numbers, dates, global attribute references)

## Known CQL parser limitations

See TODOs in `cql/grammar.peg` (and `dqd/values.go`):

- an escaped backslash (`\\`) is not accepted inside a regexp value
- a literal value (`attr=="..."`) accepts only letters, digits and escaped
  characters (e.g. `[lemma=="a.b"]` cannot be parsed)

## Remaining steps of the original plan

- step 8: numbered positions and global conditions
- step 9: coverage report on real queries (`cql2dqd -batch`)
- step 10: remaining unsupported constructs, error messages
- step 11: analysis/collocation results (ViewOptions), MCP tool `cql_to_dqd`
- NLQL integration (in progress) - see below

## NLQL integration notes

- Layer infos of LCP corpora are stored as JSON files (`<corpus>.json`) in
  a directory configured for the cqlizer MCP server (`LCP_LAYER_INFO_DIR`).
- NLQL's "DQD via CQL" mode converts the model's CQL using the MCP tool
  `cql_to_dqd` - so validation (`cql_validate`) and conversion always run
  the same cqlizer version and NLQL needs no layer infos of its own. With
  DQD as the target language, the corpus is expected to be an LCP one (no
  fallback to default props).
- Attribute names are expected to be the corpus' real names (the model
  fetches corpus info first); the only alias is `word` -> `form`.
- The cqlizer MCP tools support both Manatee and LCP corpora (routed by the
  existence of a layer info file): `get_corpus_structure` describes LCP
  corpora in terms of CQL (incl. values of categorical attributes),
  `cql_validate` checks the translation to DQD, the tagset tools refuse LCP
  corpora.
- Remaining: NLQL's corpus catalog (`find_corpora`) does not know LCP corpora
  yet.
- Layer infos are moving to NLQL (decided 2026-11-05): NLQL's DB
  (`dqd_layer_info`, filled by battery import) is the only source; battery
  runs use the layer info of the query, ad-hoc/API translations the one from
  the most recently imported battery file containing the corpus. cqlizer
  stays catalog-free - the DQD tools (`DQD_TOOLS_ENABLED`) take the layer
  info as an argument. Steps:
  1. cqlizer: DQD tools with `layer_info` (done; the former `cql_to_dqd`
     is now `dqd_translate_cql`, requires `layer_info` and is available only
     with `DQD_TOOLS_ENABLED` - the model-facing `cql_to_dqd` is NLQL's own)
  2. NLQL: latest layer info per corpus; built-in `get_corpus_structure` /
     `cql_validate` / `cql_to_dqd` routing CNC corpora to cqlizer's CNC
     tools and LCP ones to the DQD tools; cqlizer's tools hidden from the
     model (done for ad-hoc/API translations, NLQL branch
     dqd_layer_infos_from_db). Pinning the query's layer info in battery
     runs comes with DQD conversion in battery runs (see below).
     Note: layer infos imported before the "Corpus name" column have no
     corpus name, the LiRI battery must be re-imported to be found.
  3. cqlizer: remove LCP routing from the CNC tools and `LCP_LAYER_INFO_DIR`
     based lookups; the tagset
     tools then cannot tell an LCP corpus (they just won't find it)
  4. later: what to do with NLQL's `lcp-catalog.json` (`find_corpora`)
- The CQL system prompt (sysprompt_v6) hard-codes CNC defaults (syn2020,
  "word always present") - may push the model to CNC corpora with DQD as the
  target.
- Battery runs never convert to DQD (CQL is recorded and compared as CQL) -
  a LiRI battery with a CQL prompt needs `cql_to_dqd` conversion and DQD
  comparison.
