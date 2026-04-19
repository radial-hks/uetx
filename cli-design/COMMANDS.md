# 命令设计

## 一、命令总览

```
uetx <domain> <action> [flags]

Domains (v1.0 只实装 material)：
  material    Material Custom Node 相关（本 MVP 焦点）
  blueprint   [v1.1+] Blueprint 文本资产
  t3d         [v1.1+] 通用 T3D 工具（diff/validate/format）

Actions (按 domain 定义，material 当前支持)：
  generate    解析 HLSL 模板并输出 T3D Graph Snippet
  inspect     只解析模板，输出 IR（推断的 inputs / outputType）为 JSON
  validate    校验 HLSL 模板与可选 JSON 配置，返回诊断

Global:
  version     打印版本
  help        显示帮助
```

**v1.0 兼容别名**：为方便早期 Skill 集成，`uetx generate` 等价于 `uetx material generate`（其他 action 同理）。v2.0 可能移除别名，文档会提前告警。

设计原则：
- **生成与检查分离**：`generate` 产出字节串，`inspect` 产出结构化数据，`validate` 只产出诊断。
- **Skill 友好**：每个命令都支持 `--json` 把完整响应写到 stdout（单行 JSON），便于程序消费。
- **CLI 友好**：不带 `--json` 时输出人类可读文本，退出码承载成败。
- **子命令平权**：material / blueprint / t3d 共用同一套 flag 风格、退出码、IO 协议。

## 二、`generate`

### 用途
HLSL（+可选 JSON 配置）→ T3D 文本。

### 参数

| Flag | 说明 | 默认 |
|---|---|---|
| `-i, --in <path>` | HLSL 文件路径，`-` 表示 stdin | stdin |
| `-o, --out <path>` | T3D 输出路径，`-` 表示 stdout | stdout |
| `-c, --config <path>` | JSON 配置文件（覆盖解析结果） | 无 |
| `-m, --material <name>` | 材质名称 | `M_CustomNode` |
| `-t, --output-type <v>` | `CMOT_Float1..4`，覆盖模板推断 | 推断/`CMOT_Float3` |
| `-r, --route <slot>` | 追加 routing，可重复；如 `-r "Base Color" -r Opacity` | 走 defaultRouting |
| `--input <spec>` | 追加/覆盖输入，格式 `name:type[:default][:rgb]`，可重复 | 走模板解析 |
| `--json` | 以 `GenerateResponse` JSON 输出到 stdout（此时 `-o` 被忽略） | false |
| `--seed <int>` | 固定 GUID 随机种子（测试用） | 0（随机） |
| `--clipboard` | 同时把 T3D 写入系统剪贴板（跨平台） | false |
| `--no-crlf` | 以 LF 输出（仅调试，UE 要求 CRLF） | false |

### 典型用法

```bash
# 1. 文件到文件
uetx generate -i shader.hlsl -o M_Water.t3d -m M_Water

# 2. Skill 管道模式
cat shader.hlsl | uetx generate --json

# 3. 显式配置覆盖
uetx generate -i shader.hlsl -c overrides.json -o out.t3d

# 4. 混合：模板解析 + CLI 覆盖 routing
uetx generate -i shader.hlsl -r "Base Color" -r "Opacity Mask"

# 5. 手工拼输入（不用模板块）
uetx generate -i bare.hlsl \
  --input "UV:uv" \
  --input "Time:time" \
  --input "Scale:scalar:2.0" \
  --output-type CMOT_Float3 \
  -o out.t3d
```

### 配置合并优先级（低 → 高）
1. 默认值
2. HLSL 模板注释块解析结果
3. `-c` JSON 文件
4. 命令行显式 flags（`--input`, `-r`, `-t`, `-m` 等）

## 三、`inspect`

只做解析，不做 T3D 序列化。用于 Skill 先"看一眼"模板想推断出什么。

```bash
uetx inspect -i shader.hlsl [--json]
```

**文本输出**（非 `--json`）：
```
Material: M_CustomNode (default)
Output Type: CMOT_Float4  (parsed from template)
Routing: Base Color, Opacity  (default for Float4)
Inputs (5):
  [0] TexCoord      uv
  [1] TimeInput     time
  [2] GridScale     scalar  default=3.0
  [3] Speed         scalar  default=2.0
  [4] Frequency     scalar  default=40.0
Warnings: 0
```

**JSON 输出**：`GenerateResponse` 的子集，无 `t3d` 字段。

## 四、`validate`

```bash
uetx validate -i shader.hlsl [-c config.json] [--json]
```

只运行 parse + 配置合并 + 规则检查，不做 build/serialize。退出码：
- `0`：通过
- `2`：有 warning 无 error（可配 `--strict` 升级为失败）
- `1`：有 error

## 五、退出码规范

| 码 | 含义 |
|---|---|
| 0 | 成功 |
| 1 | 业务错误（解析失败、非法配置、IO 失败等） |
| 2 | 仅有 warning（仅 `validate --strict` 下出现） |
| 64 | 参数用法错误（仿 `sysexits.h` EX_USAGE） |
| 70 | 内部错误/未捕获异常 |

## 六、全局 flags

| Flag | 说明 |
|---|---|
| `--log-level <level>` | `error`/`warn`/`info`/`debug`，日志走 stderr |
| `--log-format <f>` | `text` / `json`（Skill 推荐 `json`） |
| `-q, --quiet` | 抑制 stderr |
| `-v, --version` | 等价于 `version` |

**约定**：
- **stdout 只放业务产物**（T3D 或 JSON 响应）。
- **stderr 只放日志与人类可读错误**。
- Skill 集成时默认只读 stdout。

## 七、Skill 集成建议模板

```bash
#!/usr/bin/env bash
set -euo pipefail
RESPONSE="$(uetx generate --json < "$HLSL_PATH")"
OK="$(echo "$RESPONSE" | jq -r .ok)"
if [[ "$OK" != "true" ]]; then
  echo "$RESPONSE" | jq '.errors' >&2
  exit 1
fi
echo "$RESPONSE" | jq -r .t3d > "$OUT_PATH"
```

## 八、剪贴板支持（`--clipboard`）

- Windows：调用 `clip.exe`（系统自带）
- macOS：调用 `pbcopy`
- Linux：探测 `wl-copy` / `xclip` / `xsel`，都没有则 warning 跳过

实现放在 `internal/app/clipboard.go`，失败只产生 warning，不影响主输出。
