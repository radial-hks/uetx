# uetx — Unreal Engine Text eXchange

**HLSL Template → T3D Material Graph — paste directly into Unreal Editor**

[中文文档](README_CN.md)

---

## What is uetx?

**uetx** is a Go CLI tool that converts HLSL shader templates into Unreal Engine T3D material graph snippets. The generated T3D text can be pasted directly into Unreal Editor's material graph — no manual node wiring required.

**Core pipeline:**

```
HLSL Template → Parse → IR (GraphNode / Pin / Edge) → Serialize → T3D Text
```

## Features

- **Template-driven** — Embed metadata in HLSL comments; uetx infers input types, output type, and routing automatically
- **Three usage modes** — Skill entry point (stdin/stdout JSON), local CLI for artists/TAs, batch CI processing
- **Reproducible output** — Same request + same `--seed` = byte-identical T3D
- **Cross-platform** — Single binary, zero dependencies. Builds for macOS (arm64/amd64), Windows, Linux
- **Clipboard support** — `--clipboard` to copy T3D directly via `pbcopy` / `clip.exe` / `xclip`
- **Rich diagnostics** — Structured error/warning codes (E0xx parse, E1xx config, E2xx build, W0xx warnings)

## Installation

**From source (requires Go 1.22+):**

```bash
go install github.com/radial/uetx/cmd/uetx@latest
```

**Build locally:**

```bash
git clone https://github.com/radial/uetx.git
cd uetx
go build ./cmd/uetx/
```

**Cross-compile:**

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o uetx.exe ./cmd/uetx/

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build ./cmd/uetx/
```

## Quick Start

**1. Write an HLSL template with metadata comments:**

```hlsl
/* =================================================================================
 * [Unreal Material Custom Node Template]
 * - Output Type (输出类型): [CMOT Float 3]
 *
 * Pin 0 Name: [Speed]    | Type suggestion: Scalar (Default: 2.0)
 * Pin 1 Name: [Color]    | Type suggestion: Vector 3
 * =================================================================================
 */

float3 result = Color * sin(Speed * Time);
return result;
```

**2. Generate T3D:**

```bash
uetx generate -i shader.hlsl -o MyMaterial.t3d -m M_MyShader
```

**3. Paste the contents of `MyMaterial.t3d` into Unreal Editor's material graph.**

## CLI Commands

### `uetx generate`

Convert HLSL template to T3D material graph.

```bash
# File to file
uetx generate -i shader.hlsl -o out.t3d -m M_Water

# Pipe mode (stdin → stdout)
cat shader.hlsl | uetx generate --json

# JSON request/response mode (for Skill integration)
echo '{"hlsl":"...","seed":42}' | uetx generate --stdin-json --json

# Override routing
uetx generate -i shader.hlsl -r "Base Color" -r "Opacity Mask"

# Manual inputs (no template comment block needed)
uetx generate -i bare.hlsl \
  --input "UV:uv" \
  --input "Time:time" \
  --input "Scale:scalar:2.0" \
  -t CMOT_Float3 -o out.t3d

# Copy to clipboard
uetx generate -i shader.hlsl --clipboard
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-i, --in` | Input HLSL file (`-` = stdin) | stdin |
| `-o, --out` | Output T3D file (`-` = stdout) | stdout |
| `-c, --config` | JSON config override file | — |
| `-m, --material` | Material name | `M_CustomNode` |
| `-t, --output-type` | `CMOT_Float1..4` | inferred |
| `-r, --route` | Routing slot (repeatable) | default |
| `--input` | Input spec `name:type[:default[:rgb]]` (repeatable) | parsed |
| `--json` | Output JSON response | false |
| `--stdin-json` | Read JSON request from stdin | false |
| `--seed` | Fixed GUID seed | 0 (random) |
| `--clipboard` | Copy to system clipboard | false |
| `--no-crlf` | Output LF instead of CRLF (debug) | false |

### `uetx inspect`

Parse template and show inferred metadata (no T3D output).

```bash
uetx inspect -i shader.hlsl
# Output Type: CMOT_Float4
# Routing: Base Color, Opacity
# Inputs (7):
#   - WorldPosition: vector
#   - WaterLevel: scalar
#   ...
```

### `uetx validate`

Check template + config for errors.

```bash
uetx validate -i shader.hlsl -c config.json
```

### `uetx version`

```bash
uetx version
```

## JSON API

For programmatic integration (Skill tools, CI pipelines), use `--stdin-json --json` mode.

**Request:**

```json
{
  "hlsl": "/* ... */ return float3(1,0,0);",
  "materialName": "M_MyShader",
  "outputType": "CMOT_Float3",
  "inputs": [
    { "name": "Speed", "type": "scalar", "defaultValue": "2.0" },
    { "name": "Color", "type": "vector", "useRGBMask": true }
  ],
  "routing": ["Base Color"],
  "seed": 12345
}
```

**Response (success):**

```json
{
  "ok": true,
  "t3d": "Begin Object Class=...\r\n...",
  "inferredInputs": [...],
  "effectiveOutputType": "CMOT_Float3",
  "effectiveRouting": ["Base Color"],
  "materialName": "M_MyShader",
  "stats": { "nodeCount": 5, "edgeCount": 4, "hasBreakOut": false },
  "warnings": [],
  "errors": []
}
```

**Response (error):**

```json
{
  "ok": false,
  "errors": [{ "code": "E001", "message": "HLSL input is empty" }]
}
```

## HLSL Template Format

uetx parses metadata from the **first `/* ... */` comment block** in your HLSL:

```hlsl
/*
 * - Output Type (输出类型): [CMOT Float 4]
 *
 * Pin 0 Name: [WorldPosition]   | Type suggestion: Vector 3
 * Pin 1 Name: [WaterLevel]      | Type suggestion: Scalar (Default: 0.0)
 * Pin 2 Name: [TexCoord]        | Type suggestion: TextureCoordinate
 * Pin 3 Name: [Time]            | Type suggestion: Time
 */
```

**Type inference rules** (from the "Type suggestion" field):

| Keyword in suggestion | ParamType | UE Expression |
|----------------------|-----------|---------------|
| `world` + `position` | `worldposition` | WorldPosition |
| `time` | `time` | Time |
| `texture`, `coord`, `uv` | `uv` | TextureCoordinate |
| `vector`, `color` | `vector` | VectorParameter |
| *(default)* | `scalar` | ScalarParameter |

## Architecture

```
cmd/uetx/                  ← CLI entry point
  ↓
internal/app/material/     ← Orchestration (generate / inspect / validate)
  ↓
internal/material/
  ├── parser/              ← HLSL template regex parsing, type inference
  ├── build/               ← IR construction (root, custom, params, breakout, edges)
  └── serializer/          ← T3D text output with UE escape rules
  ↓
internal/domain/           ← Pure data types (zero dependencies)
```

**Config merge priority (low → high):** built-in defaults → template parsing → JSON config file → CLI flags

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Business error (parse failure, invalid config) |
| 2 | Warnings only (validate strict mode) |
| 64 | Usage error (invalid flags) |
| 70 | Internal error |

## Error Codes

| Range | Category | Examples |
|-------|----------|----------|
| E0xx | Parse errors | E001 empty HLSL, E002 no comment block, E011 duplicate pin |
| E1xx | Config errors | E100 JSON parse fail, E110 illegal materialName chars |
| E2xx | Build errors | E200 IR construction fail |
| W0xx | Warnings | W001 no return statement, W002 zero pins parsed |

## Testing

```bash
go test ./...                                          # Run all tests
go test ./... -update                                  # Update golden files
go test ./internal/material/parser/ -run TestParseName # Single test
```

## Design Documents

Detailed specifications live in `cli-design/`:

| Document | Content |
|----------|---------|
| `CORE_KNOWLEDGE.md` | Pin tables, GUID rules, escape rules, graph topology |
| `IO_PROTOCOL.md` | JSON contracts, error codes, encoding rules |
| `COMMANDS.md` | CLI flags, config merge, exit codes |
| `ARCHITECTURE.md` | Module boundaries, type contracts |

## License

MIT
