# Stored health Sample decoding — 2026-10-24

The candidate replaces only decoding stored `health_sample.payload` rows in the statistics handler with `github.com/goccy/go-json v0.11.2`. Public request parsing, JSON serialization, session decoding and authentication keep the standard library. No JSON schema, stored payload format or error response changes are intended.

## Primary source comparison

| Pinned project | Implementation and compatibility boundary | Decision |
| --- | --- | --- |
| [goccy/go-json v0.11.2](https://github.com/goccy/go-json/releases/tag/v0.11.2), [README](https://github.com/goccy/go-json/blob/v0.11.2/README.md), [go.mod](https://github.com/goccy/go-json/blob/v0.11.2/go.mod) | Drop-in API with cached type-specific decoding; Go 1.23 minimum and no required modules of its own. [MIT license](https://github.com/goccy/go-json/blob/v0.11.2/LICENSE). | Narrow stored-Sample candidate; locally built and executed on Windows amd64 Go 1.23.2. |
| [Sonic v1.15.4](https://github.com/bytedance/sonic/releases/tag/v1.15.4), [requirements](https://github.com/bytedance/sonic/blob/v1.15.4/README.md), [configuration](https://github.com/bytedance/sonic/blob/v1.15.4/api.go), [go.mod](https://github.com/bytedance/sonic/blob/v1.15.4/go.mod) | JIT/SIMD decoder with runtime dependencies. This release supports Windows amd64 and Go 1.23. `ConfigStd` requests string copying/validation. [Apache-2.0 license](https://github.com/bytedance/sonic/blob/v1.15.4/LICENSE). | Built and measured locally, but rejected: `{"value":-0}` decodes as positive zero, changing the float bits and standard-library re-encoding in fresh and seeded actual Sample targets. |
| [easyjson v0.9.2](https://github.com/mailru/easyjson/releases/tag/v0.9.2), [README](https://github.com/mailru/easyjson/blob/v0.9.2/README.md), [go.mod](https://github.com/mailru/easyjson/blob/v0.9.2/go.mod) | Go 1.20 minimum; generates struct-specific code and does not promise full standard-library equivalence. [MIT license](https://github.com/mailru/easyjson/blob/v0.9.2/LICENSE). | Researched, not benchmarked. This change would require generated methods on the production type or a duplicated DTO contract. |

The dependency carries its upstream implementation under MIT; this note records its attribution/license source. Sonic and easyjson are research references only and are not production dependencies.

## Compatibility evidence

The isolated experiment imports the actual production `health.Sample`, rather than copying a substitute type. It compares 317 inputs with fresh and populated decode targets: 256 stored-like representations, complete optional/max-size fields, nullable values/maps, duplicate/case-insensitive/escaped keys, integer/float bounds, malformed/trailing data, invalid UTF-8/surrogates, unknown fields and time parsing. Successful fields, canonical standard-library output and float64 bits match; failure acceptance also matches. Separate probes overwrite input bytes after decoding and verify that scalar strings and metadata keys/values remain unchanged.

The committed regression file `internal/handlers/stored_sample_decoder_test.go` retains this corpus and additionally varies finite float64 bit patterns across the generated samples. It compares failure acceptance rather than exact error text or partial output: the stored-row handler discards every decoding failure and returns the same generic error response. This finite corpus does not establish exhaustive equivalence.

## Serial decoder measurements

Windows amd64, Go 1.23.2, GOAMD64=v1, AMD Ryzen AI 9 H 365. Independent decoder benchmark, `-benchtime=200ms -count=3`; table shows medians. These are synthetic stored payloads and do not measure SQL, network, cache or production endpoint latency.

| Actual Sample shape | encoding/json | go-json v0.11.2 | Sonic ConfigStd |
| --- | ---: | ---: | ---: |
| Sparse | 1,977 ns | 343.2 ns | 646.3 ns |
| Optional fields + UTF-8 metadata | 3,629 ns | 1,062 ns | 1,190 ns |
| Maximum fields/metadata | 1,046,836 ns | 419,798 ns | 106,670 ns |
| 1,000 mixed stored samples | 2,405,868 ns | 543,032 ns | 917,225 ns |

For 1,000 mixed samples, go-json reduced measured decoder time by 77.4%, allocated bytes from approximately 719 KB to 436 KB (39.3%), and allocations from 16,733 to 2,532 (84.9%). Sparse/mixed/max-field allocation counts were 12/26/218 with the standard library and 2/4/71 with go-json. Sonic was fastest on maximum metadata, but allocated approximately 589 KB per decode against standard 295 KB and go-json 291 KB, and failed the signed-zero compatibility check. Gains varied with payload shape; no endpoint gain is inferred from these microbenchmarks.

The original decoder timing and compatibility output is committed in [evidence/decoder-experiment.txt](evidence/decoder-experiment.txt), including the rejected Sonic failure. Integration results and limits are in [README.md](README.md). Reproduce the focused committed compatibility checks from the backend directory with:

```powershell
go test ./internal/handlers -run '^TestStoredSampleDecoder' -count=1
```


The selected dependency license is retained verbatim in [goccy-go-json-LICENSE.txt](../../licenses/goccy-go-json-LICENSE.txt).
