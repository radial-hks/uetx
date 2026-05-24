# uetx Fix Specs — 3-Way Review Aggregation

## Review Summary

Cross-referenced 3 parallel Claude Code reviews (Go correctness, Architecture, Test coverage).
Total cost: ~$3.66. Findings consolidated by priority and deduplicated.

---

## P0: Must Fix (wrong output / crash / zero test coverage)

### Task C0-1: Fix BreakOut multi-scalar channel routing
**File:** internal/material/build/build.go ~line 122-130
**Bug:** When outputType=Float4 and routing has 2+ scalar slots (e.g. ["Opacity", "Metallic"]),
ALL scalar slots wire to `breakOutPinMap["A"]` (alpha channel). Result: Opacity and Metallic get the same value.
**Fix:** Map each scalar slot to a different channel. Suggested: assign channels R→G→B→A in routing order,
or use a deterministic slot-specific mapping.
**Do NOT break:** single-scalar routing, existing Float3/2/1 paths, BreakOut creation logic.
**Verify:** Test with routing=["Opacity","Metallic","Ambient Occlusion"] — check 3 distinct channels in output.

### Task C0-2: Add cmd/uetx integration tests
**File:** NEW cmd/uetx/main_test.go
**Gap:** Zero test coverage for the CLI layer. No tests for flag parsing, subcommand dispatch, JSON protocol.
**Fix:** Table-driven tests that:
- Run `go run ./cmd/uetx/` with testdata fixtures via stdin
- Verify exit codes (0/1/2/64/70)
- Verify JSON output shape for generate/inspect/validate
- Test --seed flag for reproducible output
- Test config merge priority (CLI flag overrides JSON over HLSL defaults)
**Verify:** `go test ./cmd/uetx/ -v` passes; at minimum 5 test cases.

### Task C0-3: Deepen golden comparison
**File:** internal/app/material/generate_test.go (or NEW if needed)
**Gap:** Golden comparison checks node count + type set but NOT LinkedTo graph isomorphism.
Two graphs with same node types but different wiring would pass.
**Fix:** Add graph isomorphism check: compare edge sets (from/to pairs), code content match,
output type match. The comparison should be structural (ignore GUIDs).
**Verify:** Add a deliberate wire-swap test case that the deepened check catches.

### Task C0-4: Fix escapeHLSL bare CR handling
**File:** internal/material/build/build.go ~line 319-326 (escapeHLSL function)
**Bug:** Replacer covers `\r\n` and `\n`, but lone `\r` (legacy Mac / stray CR) passes through
and may break UE's parser.
**Fix:** Add `"\r" → "\\r\n"` as the last replacer step.
**Verify:** Test with input containing bare `\r` — output should have `\r\n` only.

---

## P1: Likely Bugs

### Task C1-1: Fix W005 diagnostic meaning
**File:** internal/app/material/generate.go ~line 64-69
**Bug:** W005 emitted for "empty routing defaults". Per CORE_KNOWLEDGE.md §12, W005 is reserved for
"routing contains ScalarSlot but outputType ≠ Float4". The real W005 condition is never checked.
**Fix:** Rename current W005 to a different warning code (e.g. W008), then implement actual W005:
emit when routing has scalar slot entries but request.OutputType != CMOTFloat4.
**Verify:** Test case: Float3 output with routing to scalar slot → W005 fires.

### Task C1-2: Unknown routing slots → diagnostic
**File:** internal/material/build/build.go ~line 118-121
**Bug:** Unknown routing slots are silently dropped. `findRootPin` returns nil, loop continues.
**Fix:** Append a W0xx diagnostic when a routing slot doesn't match any RootPinTable entry.
**Verify:** Test with routing=["FooBar"] → diagnostic emitted, no crash.

### Task C1-3: Fix CLI RGB-mask flag parsing
**File:** cmd/uetx/main.go ~line 414
**Bug:** Parses `--input name:type:default:rgb` by checking `EqualFold(parts[3], "true")`.
The documented form is literal `:rgb` (not `:true`). Current behavior: `:rgb` silently
fails to set UseRGBMask.
**Fix:** Accept both `:rgb` and `:true` as valid RGB-mask indicators. Or align with docs:
`EqualFold(parts[3], "rgb")`.
**Verify:** Test `--input foo:vector:1,0,0,1:rgb` → UseRGBMask=true.

### Task C1-4: Surface scalar/vector default parse errors
**File:** internal/material/build/build.go ~line 355-360 (parseScalarDefault, parseVectorDefault)
**Bug:** Parse errors silently fall back to 0 / empty. No diagnostic surfaced.
**Fix:** Return error or emit W0xx diagnostic on malformed defaults.
**Verify:** Test with `(Default: "not-a-number")` → warning emitted.

### Task C1-5: applyEdges error handling
**File:** internal/material/build/build.go ~line 347-352
**Bug:** nil fromPin/toPin silently skips — this is an invariant violation (edge list vs node pins mismatch).
**Fix:** Panic in dev mode or return error. At minimum log.
**Verify:** N/A (panics are test-time). Add comment explaining this is an invariant guard.

### Task C1-6: Panic guard on empty diagnostic code
**File:** internal/app/material/generate.go ~line 40, 88, 124
**Bug:** `d.Code[0]` panics if Code is empty string. Use `strings.HasPrefix(d.Code, "E")` or `len(d.Code) > 0` guard.
**Fix:** Add length guard before indexing d.Code[0].
**Verify:** Test with Diagnostic{Code: ""} → no panic.

### Task C1-7: Comment block regex picks first block
**File:** internal/material/parser/template.go ~line 19
**Bug:** If HLSL has multiple `/* ... */` blocks and template metadata isn't in the first one,
the wrong block is parsed.
**Fix:** Prefer the longest block, or the first block containing a `Pin` line.
**Verify:** Test with two comment blocks, metadata in second one.

---

## P2: Minor / Optimization

### Task C2-1: MaterialOutputSlot → distinct type
**File:** internal/domain/types.go:25
**Issue:** `MaterialOutputSlot` is a type *alias* (`= string`), not a distinct named type.
ARCHITECTURE.md specifies a distinct type for compile-time safety.
**Fix:** Change `type MaterialOutputSlot = string` to `type MaterialOutputSlot string`.
Update all callers (may need string() conversion in a few places).
**Verify:** `go build ./...` passes.

### Task C2-2: Implement missing warnings (W001, W003, W004)
**File:** internal/app/material/generate.go
**Gap:** Per CORE_KNOWLEDGE.md §12:
- W001: HLSL missing `return` statement
- W003: Pin name vs code-body variable mismatch
- W004: useRGBMask=true on non-vector type (silently dropped at build.go:244)
**Fix:** Add these diagnostics. W004 is simplest: check param type before dropping mask.
**Verify:** Each warning has a test case that triggers it.

### Task C2-3: Remove dead parameters in buildParams
**File:** internal/material/build/build.go:81, 171
**Issue:** `buildParams` accepts `customGraphName` and `customExprName` params but never uses them.
**Fix:** Remove from signature. Update callers.
**Verify:** `go build ./...` passes.

### Task C2-4: Package-level regex precompilation
**File:** internal/material/parser/template.go:104-106
**Issue:** `normalizeOutputType` compiles two regexes on every call.
**Fix:** Compile once at package level (var).
**Verify:** Tests still pass.

### Task C2-5: Deprecated mrand.Read → binary encoding
**File:** internal/domain/guid.go:28
**Issue:** `mrand.Read` deprecated since Go 1.22.
**Fix:** Use `binary.LittleEndian.PutUint64` from `r.Uint64()`.
**Verify:** Deterministic GUID output with same seed.

### Task C2-6: Add idempotency test
**File:** NEW test (place in appropriate *_test.go)
**Issue:** Same request + same seed must produce byte-identical output. Under-asserted.
**Fix:** Run generate twice with same seed, compare output byte-for-byte.
**Verify:** Test passes.

### Task C2-7: T3D trailing CRLF check
**File:** internal/material/serializer/t3d.go:13-19
**Issue:** Final "End Object" has no trailing CRLF. Check against golden baseline.
**Fix:** If golden has trailing CRLF, append one.
**Verify:** Golden comparison catches mismatch.

---

## Batch Execution Plan

### Batch 1 (P0 — 4 fixes):
C0-1 (BreakOut channels) + C0-2 (CLI tests) + C0-3 (golden comparison) + C0-4 (bare CR escape)
All target different files. Run in parallel (3 instances):
- Instance A: C0-1 + C0-4 (both in build.go, same file)
- Instance B: C0-2 (new file cmd/uetx/main_test.go)
- Instance C: C0-3 (generate_test.go)

### Batch 2 (P1 — 7 fixes):
C1-1 through C1-7. Some touch same files as Batch 1, so run sequentially after Batch 1.

### Batch 3 (P2 — 7 fixes):
C2-1 through C2-7.

---

DO NOT:
- Change RootPinTable (30 entries is correct, docs are wrong not code)
- Change BreakOut trigger logic (NeedsBreakOut is correct)
- Modify domain types except C2-1
- Remove any functionality

ALWAYS:
- Run `go build ./...` and `go test ./...` after each batch
- Commit per batch with conventional commit message (e.g., "fix: P0 BreakOut channel routing")
