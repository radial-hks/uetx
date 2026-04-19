# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**uetx** (Unreal Engine Text eXchange) is a Go CLI tool and SDK for bidirectional conversion of Unreal Engine text-based assets. The v1.0 MVP focuses on converting HLSL shader templates into T3D material graph snippets that can be pasted into Unreal Editor.

Core pipeline: **HLSL Template → Parse → IR (GraphNode/Pin/Edge) → Serialize → T3D text**

Three usage modes: Skill entry point (stdin/stdout JSON), local CLI for artists/TAs, batch CI processing.

## Build & Test Commands

```bash
# Build
go build ./cmd/uetx/

# Run tests
go test ./...

# Run a single test
go test ./internal/material/parser/ -run TestParseName

# Run tests with golden file update
go test ./... -update

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o uetx.exe ./cmd/uetx/

# Quick version check
go run ./cmd/uetx/ version
```

## Architecture

Strict one-way dependency hierarchy (bottom-up only):

```
cmd/uetx/              ← CLI entry (Cobra or std flag)
  ↓
internal/app/          ← Application layer (config merge, IO, diagnostics)
  ↓
parser/ + build/ + serializer/  ← Business logic
  ↓
domain/                ← Pure data types (Pin, Node, Edge) — zero dependencies
```

Key directories:
- `cmd/uetx/` — CLI dispatcher, flag parsing, exit codes
- `internal/app/material/` — generate/inspect/validate workflows
- `internal/material/parser/` — HLSL template regex parsing, type inference
- `internal/material/build/` — IR construction (root, custom, params, breakout, edges)
- `internal/material/serializer/` — T3D text output with UE escape rules
- `domain/` — Core types: `GraphNode`, `Pin`, `Edge`, `ParamType`, `OutputType`
- `pkg/uetx/` — Public stable API (semver-locked)
- `testdata/material/` — Golden files and test fixtures

## Key Design Documents

All design specs live in `cli-design/`:
- **CORE_KNOWLEDGE.md** — Authoritative domain rules (pin tables, GUID format, escape rules, graph topology). **Read this before modifying any business logic.**
- **IO_PROTOCOL.md** — JSON request/response contracts, error codes (E0xx/E1xx/E2xx/W0xx), line encoding rules
- **COMMANDS.md** — CLI flags, config merge priority, exit codes
- **ARCHITECTURE.md** — Module boundaries, type contracts, testing strategy

Reference implementations in `cli-design/reference/`:
- `T3DGenerator.ts` — Oracle for T3D serialization logic
- `UENodeGenerator.tsx` — Oracle for template parsing
- `M_WaterLevel_unrealeditor.txt` — Golden baseline (real UE export)
- `M_WaterLevel_webeditor.txt` — Sample HLSL input template

## Critical Domain Rules

- **T3D output must use CRLF** (`\r\n`) — Unreal Engine requirement
- **Root node has exactly 30 pins** in a fixed order — see CORE_KNOWLEDGE.md RootPinTable (docs say 34 but actual table and golden baseline both have 30)
- **GUIDs**: 32 uppercase hex chars, no hyphens. `PersistentGuid` is always all-zeros. Support `--seed` for reproducible output.
- **BreakOut node**: Required when outputType=Float4 AND any routing target is a scalar-only slot (Opacity, Metallic, etc.)
- **Pin serialization**: One long line per pin, 3-space indent, specific field ordering
- **Custom node Code field**: HLSL with backslash-escaped CRLF, escaped quotes
- **Config merge priority** (low→high): defaults → HLSL template parsing → JSON config file → CLI flags
- **Idempotency**: Same request + same seed must produce byte-identical output

## Testing Strategy

- **Parser tests**: table-driven, text → NodeInput[]
- **Build tests**: snapshot, GenerateRequest → []*GraphNode
- **Serializer tests**: golden file comparison, []*GraphNode → T3D string
- **App tests**: end-to-end with IO boundary
- **Golden comparison is structural** (not byte-exact) because GUIDs vary — compare node counts/types, pin sets, connection graph isomorphism, code content, and output types

## Exit Codes

- `0` success, `1` business error, `2` warnings-only (strict mode), `64` usage error, `70` internal error
