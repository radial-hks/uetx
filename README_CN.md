# uetx — Unreal Engine Text eXchange

**HLSL 模板 → T3D 材质图 — 直接粘贴到 Unreal Editor**

[English Documentation](README.md)

---

## 什么是 uetx？

**uetx**（Unreal Engine Text eXchange）是一个 Go 命令行工具，用于将 HLSL 着色器模板转换为 Unreal Engine T3D 材质图代码片段。生成的 T3D 文本可以直接粘贴到 Unreal Editor 的材质图中，无需手动连线。

**核心流水线：**

```
HLSL 模板 → 解析 → 中间表示 (GraphNode / Pin / Edge) → 序列化 → T3D 文本
```

## 功能特性

- **模板驱动** — 在 HLSL 注释中嵌入元数据，uetx 自动推断输入类型、输出类型和路由
- **三种使用模式** — Skill 入口（stdin/stdout JSON）、本地 CLI（面向美术/TA）、批量 CI 处理
- **可复现输出** — 相同请求 + 相同 `--seed` = 字节级一致的 T3D 输出
- **跨平台** — 单一二进制文件，零依赖。支持 macOS (arm64/amd64)、Windows、Linux
- **剪贴板支持** — `--clipboard` 通过 `pbcopy` / `clip.exe` / `xclip` 直接复制 T3D
- **丰富的诊断信息** — 结构化错误/警告码（E0xx 解析、E1xx 配置、E2xx 构建、W0xx 警告）

## 安装

**从源码安装（需要 Go 1.22+）：**

```bash
go install github.com/radial/uetx/cmd/uetx@latest
```

**本地构建：**

```bash
git clone https://github.com/radial/uetx.git
cd uetx
go build ./cmd/uetx/
```

**交叉编译：**

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o uetx.exe ./cmd/uetx/

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build ./cmd/uetx/
```

## 快速开始

**1. 编写带元数据注释的 HLSL 模板：**

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

**2. 生成 T3D：**

```bash
uetx generate -i shader.hlsl -o MyMaterial.t3d -m M_MyShader
```

**3. 将 `MyMaterial.t3d` 的内容粘贴到 Unreal Editor 的材质图中。**

## CLI 命令

### `uetx generate`

将 HLSL 模板转换为 T3D 材质图。

```bash
# 文件到文件
uetx generate -i shader.hlsl -o out.t3d -m M_Water

# 管道模式（stdin → stdout）
cat shader.hlsl | uetx generate --json

# JSON 请求/响应模式（用于 Skill 集成）
echo '{"hlsl":"...","seed":42}' | uetx generate --stdin-json --json

# 自定义路由
uetx generate -i shader.hlsl -r "Base Color" -r "Opacity Mask"

# 手动指定输入（无需模板注释块）
uetx generate -i bare.hlsl \
  --input "UV:uv" \
  --input "Time:time" \
  --input "Scale:scalar:2.0" \
  -t CMOT_Float3 -o out.t3d

# 复制到剪贴板
uetx generate -i shader.hlsl --clipboard
```

**参数：**

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-i, --in` | 输入 HLSL 文件（`-` = stdin） | stdin |
| `-o, --out` | 输出 T3D 文件（`-` = stdout） | stdout |
| `-c, --config` | JSON 配置覆盖文件 | — |
| `-m, --material` | 材质名称 | `M_CustomNode` |
| `-t, --output-type` | `CMOT_Float1..4` | 自动推断 |
| `-r, --route` | 路由插槽（可重复） | 默认路由 |
| `--input` | 输入规格 `名称:类型[:默认值[:rgb]]`（可重复） | 解析结果 |
| `--json` | 输出 JSON 响应 | false |
| `--stdin-json` | 从 stdin 读取 JSON 请求 | false |
| `--seed` | 固定 GUID 种子 | 0（随机） |
| `--clipboard` | 复制到系统剪贴板 | false |
| `--no-crlf` | 输出 LF 而非 CRLF（调试用） | false |

### `uetx inspect`

解析模板并显示推断的元数据（不生成 T3D）。

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

检查模板和配置是否有错误。

```bash
uetx validate -i shader.hlsl -c config.json
```

### `uetx version`

```bash
uetx version
```

## JSON API

用于程序化集成（Skill 工具、CI 流水线），使用 `--stdin-json --json` 模式。

**请求：**

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

**响应（成功）：**

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

**响应（错误）：**

```json
{
  "ok": false,
  "errors": [{ "code": "E001", "message": "HLSL input is empty" }]
}
```

## HLSL 模板格式

uetx 从 HLSL 中**第一个 `/* ... */` 注释块**解析元数据：

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

**类型推断规则**（基于 "Type suggestion" 字段）：

| 建议中的关键词 | ParamType | UE 表达式 |
|---------------|-----------|-----------|
| `world` + `position` | `worldposition` | WorldPosition |
| `time` | `time` | Time |
| `texture`、`coord`、`uv` | `uv` | TextureCoordinate |
| `vector`、`color` | `vector` | VectorParameter |
| *（默认）* | `scalar` | ScalarParameter |

## 架构

```
cmd/uetx/                  ← CLI 入口
  ↓
internal/app/material/     ← 编排层（generate / inspect / validate）
  ↓
internal/material/
  ├── parser/              ← HLSL 模板正则解析、类型推断
  ├── build/               ← IR 构建（root、custom、params、breakout、edges）
  └── serializer/          ← T3D 文本输出与 UE 转义规则
  ↓
internal/domain/           ← 纯数据类型（零依赖）
```

**配置合并优先级（低 → 高）：** 内置默认值 → 模板解析 → JSON 配置文件 → CLI 参数

## 退出码

| 码 | 含义 |
|----|------|
| 0 | 成功 |
| 1 | 业务错误（解析失败、配置无效） |
| 2 | 仅警告（validate 严格模式） |
| 64 | 用法错误（无效参数） |
| 70 | 内部错误 |

## 错误码

| 范围 | 类别 | 示例 |
|------|------|------|
| E0xx | 解析错误 | E001 HLSL 为空、E002 无注释块、E011 重复引脚名 |
| E1xx | 配置错误 | E100 JSON 解析失败、E110 materialName 含非法字符 |
| E2xx | 构建错误 | E200 IR 构建失败 |
| W0xx | 警告 | W001 无 return 语句、W002 解析到 0 个引脚 |

## 测试

```bash
go test ./...                                          # 运行所有测试
go test ./... -update                                  # 更新 golden 文件
go test ./internal/material/parser/ -run TestParseName # 单个测试
```

## 设计文档

详细规格说明在 `cli-design/` 目录中：

| 文档 | 内容 |
|------|------|
| `CORE_KNOWLEDGE.md` | 引脚表、GUID 规则、转义规则、图拓扑 |
| `IO_PROTOCOL.md` | JSON 契约、错误码、编码规则 |
| `COMMANDS.md` | CLI 参数、配置合并、退出码 |
| `ARCHITECTURE.md` | 模块边界、类型契约 |

## 许可证

MIT
