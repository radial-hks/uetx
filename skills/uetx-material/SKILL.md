---
name: uetx-material
description: Describe a shader effect in natural language → generate HLSL template → convert to T3D material graph for Unreal Engine
trigger: /uetx
---

# /uetx — Natural Language → Unreal Material Graph

Turn a natural-language shader description into a pasteable Unreal Engine T3D material graph.

**Pipeline:** User Description → HLSL Template → `uetx generate` → T3D Text

## Role

You are an expert Unreal Engine Technical Artist and Shader Programmer. When this skill is invoked, you will:

1. Interpret the user's shader effect description
2. Generate an HLSL Custom Node template following the strict formatting rules below
3. Feed the template to `uetx generate` to produce a T3D material graph
4. Present the results to the user

## HLSL Template — Critical Formatting Rules

These rules are **mandatory**. Violations will cause `uetx` to fail parsing.

### Rule 1: Pin Naming Syntax
Every input variable name in the INPUTS section MUST be enclosed in square brackets `[]`.
- Correct: `Pin 0 Name: [Time]`
- Wrong: `Pin 0 Name: Time`

The variable name in the code body must match the Pin Name exactly (without brackets).

### Rule 2: Output Type Syntax
The Output Type value MUST be enclosed in square brackets `[]`.
- Correct: `- Output Type (输出类型): [CMOT Float 3]`
- Wrong: `- Output Type (输出类型): CMOT Float 3`

Valid output types: `CMOT Float 1`, `CMOT Float 2`, `CMOT Float 3`, `CMOT Float 4`

### Rule 3: Type Suggestion Precision
Use ONLY these standard Unreal Engine Material Editor pin types:

| HLSL Type | Type Suggestion |
|-----------|----------------|
| `float` | Scalar |
| `float2` | Vector 2 or TextureCoordinate |
| `float3` | Vector 3 |
| `float4` | Vector 4 |
| `Texture2D` | Texture Object |

Additional special types recognized by `uetx`:
- **Time** — maps to UE Time node
- **WorldPosition** (or "Vector 3" with keyword "world position") — maps to WorldPosition node

### Rule 4: Language
- The `Description` and comments within the template header must be in **Chinese (Simplified)**
- The code logic itself must be standard HLSL

### Rule 5: Default Values
- Scalar defaults: a single number (e.g., `(Default: 2.0)`)
- Vector defaults: comma-separated (e.g., `(Default: 1.0, 0.5, 0.0)`)
- Defaults are optional — omit if no sensible default exists

## Universal Template Format

```hlsl
/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): [简短中文描述]
 * - Output Type (输出类型): [CMOT Float N]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [InputName0]     | Type suggestion: TypeHint (Default: X)
 * Pin 1 Name: [InputName1]     | Type suggestion: TypeHint (Default: X)
 * [Add more pins as needed...]
 * =================================================================================
 */

// --- [CODE BODY START] ---

// Step 1: ...
[HLSL code]

// Step N: Return
return [Result];

// --- [CODE BODY END] ---
```

## Execution Steps

When the user invokes `/uetx <description>`, execute these steps:

### Step 1: Understand the Request
Parse the user's natural-language description. Identify:
- What visual effect they want
- Required inputs (textures, parameters, coordinates)
- Expected output dimensionality (float, float3, float4)
- Any routing preferences (e.g., "connect to Opacity")

### Step 2: Generate HLSL Template
Write an HLSL template following the Universal Template Format above. Ensure:
- All formatting rules are strictly followed
- Pin names match variable names in the code body
- Output type matches the return type
- Code is optimized and uses standard UE Custom Node patterns
- Comments in the header are in Chinese

### Step 3: Pre-check and Convert

First, verify `uetx` is available:

```bash
which uetx || echo "ERROR: uetx not found in PATH. Install: go install github.com/radial/uetx/cmd/uetx@latest"
```

If `uetx` is not found, stop and tell the user how to install it (see FAQ.md for platform-specific instructions).

Then save the HLSL template and run conversion:
Save the HLSL template to a temporary `.hlsl` file and run `uetx generate`:

```bash
# Write HLSL to temp file
cat > /tmp/uetx_skill_temp.hlsl << 'HLSL_EOF'
<generated HLSL template>
HLSL_EOF

# Generate T3D via JSON API (--json-out writes UTF-8 JSON to file, -o writes T3D)
uetx generate -i /tmp/uetx_skill_temp.hlsl --json
```

If the user specified a material name, add `-m <name>`.
If the user specified routing, add `-r "<slot>"` for each slot.
If the user wants clipboard output, add `--clipboard`.
If the user wants file output, add `-o <path>`.
If the user wants both T3D file and JSON metadata, use `-o <path>.t3d --json-out <path>.json` (both work simultaneously).
If the user wants all artifacts in one directory, use `--artifact-dir <dir>` to produce `output.t3d`, `generate.json`, and `effective-config.json`.

### Step 4: Handle Response
Parse the JSON response:

**On success** (`"ok": true`):
- Show the generated HLSL template (in a code block)
- Show T3D stats: node count, edge count, breakout presence
- If there are warnings, display them with hints
- Show where the T3D was saved or if it was copied to clipboard
- If no output file was specified, offer to save or copy

**On failure** (`"ok": false`):
- Show the error codes and messages
- Diagnose the issue (likely a template formatting problem)
- Fix the HLSL template and retry automatically

### Step 5: Cleanup
Remove the temporary HLSL file after successful conversion.

## Argument Parsing

The user's input after `/uetx` can include optional flags mixed with the description:

| Pattern | Meaning | Example |
|---------|---------|---------|
| `-m <name>` | Material name | `/uetx -m M_Water create water ripples` |
| `-o <path>` | Save T3D to file | `/uetx -o water.t3d water shader` |
| `-r "<slot>"` | Route to material slot | `/uetx -r "Opacity" a dissolve effect` |
| `--clipboard` | Copy T3D to clipboard | `/uetx --clipboard fire effect` |
| `--seed <N>` | Fixed seed for reproducibility | `/uetx --seed 42 noise pattern` |
| `--json-out <path>` | Write JSON response to file (UTF-8) | `/uetx --json-out result.json fire effect` |
| `--artifact-dir <dir>` | Write all artifacts to directory | `/uetx --artifact-dir ./out fire effect` |

If no flags are provided, use defaults: material name `M_CustomNode`, output to stdout, default routing.

## Error Code Reference

| Code | Meaning | Common Fix |
|------|---------|------------|
| E001 | HLSL input is empty | Template generation failed — retry |
| E002 | No comment block found | Missing `/* ... */` header — fix template |
| E011 | Duplicate pin name | Two pins share the same name — rename one |
| E020 | Invalid output type | Must be CMOT_Float1..4 — fix template |
| E101 | Unknown output type | Use CMOT_Float1, CMOT_Float2, CMOT_Float3, or CMOT_Float4 |
| E102 | Unknown routing slot | Check slot name against UE Root pin table |
| E103 | Unknown input type | Use scalar, vector, time, uv, or worldposition |
| E110 | Illegal material name | Name must be `[A-Za-z0-9_]` only |
| W001 | No return statement | Add `return` to code body |
| W002 | Zero pins parsed | Pin format not matching — check `[]` brackets |

## Examples

### Example 1: Simple Fresnel Edge Glow
```
/uetx create a Fresnel-based edge glow effect, output float3 color
```

Expected HLSL output:
```hlsl
/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): [基于菲涅尔的边缘发光效果]
 * - Output Type (输出类型): [CMOT Float 3]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [Power]       | Type suggestion: Scalar (Default: 3.0)
 * Pin 1 Name: [GlowColor]   | Type suggestion: Vector 3 (Default: 0.2, 0.5, 1.0)
 * Pin 2 Name: [Intensity]    | Type suggestion: Scalar (Default: 2.0)
 * =================================================================================
 */

// --- [CODE BODY START] ---

// 1. Calculate Fresnel term
float3 worldNormal = normalize(Parameters.WorldNormal);
float3 cameraDir = normalize(Parameters.CameraVector);
float fresnel = pow(1.0 - saturate(dot(worldNormal, cameraDir)), Power);

// 2. Apply glow color and intensity
float3 result = GlowColor * fresnel * Intensity;

return result;

// --- [CODE BODY END] ---
```

### Example 2: UV Scrolling with Time
```
/uetx -m M_Lava --clipboard scrolling UV effect for lava texture
```

### Example 3: Dissolve Effect with Opacity
```
/uetx -r "Opacity" -r "Base Color" dissolve effect using noise threshold
```
