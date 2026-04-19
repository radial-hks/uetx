# 架构与目录结构

## 一、模块划分（依赖方向单向）

```
                 ┌──────────────┐
                 │   cmd/       │  CLI 入口（Cobra 或 std flag）
                 └──────┬───────┘
                        │
                        ▼
                 ┌──────────────┐
                 │  internal/   │  应用层：配置合并、IO、诊断
                 │   app/       │
                 └──────┬───────┘
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │ parser/  │  │  core/   │  │serializer│
   │ template │  │ IR build │  │   t3d/   │
   └──────────┘  └──────────┘  └──────────┘
                        │
                        ▼
                 ┌──────────────┐
                 │  domain/     │  纯数据类型：Pin / Node / Edge / Input
                 └──────────────┘
```

**依赖规则**：下层不依赖上层。`domain` 零依赖。`parser/serializer/core` 只依赖 `domain`。`app` 组合三者。`cmd` 只依赖 `app`。

## 二、推荐目录结构（新仓库）

> 目录按 **domain（material / blueprint / t3d）** 分目，v1.0 只实装 `material`，其他目录预留占位但不必创建空包。`parser/serializer/core` 都在 domain 内自洽，顶层 `domain/` 放跨域共享类型。

```
uetx/
├── cmd/
│   └── uetx/
│       └── main.go              # CLI 入口：子命令分发 material/blueprint/t3d
│
├── internal/
│   ├── app/
│   │   ├── material/
│   │   │   ├── generate.go      # material generate：parse + build + serialize
│   │   │   ├── inspect.go       # material inspect：只解析，返回 IR JSON
│   │   │   └── validate.go      # material validate：只校验
│   │   ├── blueprint/           # [v1.1+] 预留
│   │   ├── t3d/                 # [v1.1+] 预留
│   │   └── diagnostics.go       # 统一诊断聚合（warnings/errors）
│   │
│   ├── domain/                  # 跨域共享的最基础类型
│   │   ├── guid.go              # GenerateGUID（32-hex 大写）
│   │   └── diagnostic.go        # Diagnostic / Code 常量
│   │
│   ├── material/                # Material 域：解析 + IR + 序列化
│   │   ├── types.go             # ParamType, OutputType, MaterialOutputSlot, NodeInput
│   │   ├── tables.go            # ROOT_PIN_TABLE, SCALAR_SLOTS（详见 CORE_KNOWLEDGE）
│   │   ├── parser/
│   │   │   ├── template.go      # Pin X Name / Output Type 正则解析
│   │   │   ├── typeinfer.go     # Type suggestion → ParamType 推断
│   │   │   └── template_test.go
│   │   ├── build/
│   │   │   ├── build.go         # BuildIR：Root + Custom + Params + BreakOut
│   │   │   ├── routing.go       # defaultRouting + needsBreakOut
│   │   │   ├── edges.go         # ApplyEdges：双向 LinkedTo 回填
│   │   │   └── build_test.go
│   │   └── serializer/
│   │       ├── t3d.go           # SerializeGraph：节点 + Pin 序列化
│   │       ├── escape.go        # HLSL Code 字符串转义
│   │       └── t3d_test.go
│   │
│   ├── blueprint/               # [v1.1+] 预留：蓝图域
│   └── t3d/                     # [v1.1+] 预留：通用 T3D 解析/格式化
│
├── pkg/                         # 对外稳定 API（Semver 锁）
│   └── uetx/
│       ├── material.go          # Material.Generate(req) (resp, error)
│       ├── blueprint.go         # [v1.1+]
│       └── t3d.go               # [v1.1+]
│
├── testdata/
│   ├── material/
│   │   ├── golden/
│   │   │   ├── M_WaterLevel.hlsl
│   │   │   └── M_WaterLevel.t3d # 对拍基线
│   │   ├── templates/
│   │   │   ├── raindrop.hlsl
│   │   │   └── breathing.hlsl
│   │   └── fixtures/
│       └── *.json               # 各种配置样例
│
├── docs/
│   ├── CORE_KNOWLEDGE.md        # 从本设计包拷贝过去
│   ├── IO_PROTOCOL.md
│   └── CHANGELOG.md
│
├── scripts/
│   ├── build-all.sh             # 多平台交叉编译
│   └── release.sh
│
├── .github/workflows/
│   ├── ci.yml                   # go test + vet + staticcheck
│   └── release.yml              # goreleaser 打多平台包
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

## 三、核心类型契约（Go 版）

```go
// domain/types.go
package domain

type ParamType string
const (
    ParamScalar        ParamType = "scalar"
    ParamVector        ParamType = "vector"
    ParamTime          ParamType = "time"
    ParamUV            ParamType = "uv"
    ParamWorldPosition ParamType = "worldposition"
)

type OutputType string
const (
    CMOTFloat1 OutputType = "CMOT_Float1"
    CMOTFloat2 OutputType = "CMOT_Float2"
    CMOTFloat3 OutputType = "CMOT_Float3"
    CMOTFloat4 OutputType = "CMOT_Float4"
)

type MaterialOutputSlot string // "Base Color", "Opacity", ...

type NodeInput struct {
    Name         string    `json:"name"`
    Type         ParamType `json:"type"`
    DefaultValue string    `json:"defaultValue,omitempty"`
    UseRGBMask   bool      `json:"useRGBMask,omitempty"`
}

type GenerateRequest struct {
    HLSL         string               `json:"hlsl"`
    MaterialName string               `json:"materialName,omitempty"` // 默认 "M_CustomNode"
    OutputType   OutputType           `json:"outputType,omitempty"`   // 不填走解析/默认
    Inputs       []NodeInput          `json:"inputs,omitempty"`       // 不填走解析
    Routing      []MaterialOutputSlot `json:"routing,omitempty"`      // 不填走 defaultRouting
    Seed         int64                `json:"seed,omitempty"`         // GUID 可复现种子，测试用
}

type GenerateResponse struct {
    OK              bool           `json:"ok"`
    T3D             string         `json:"t3d,omitempty"`
    InferredInputs  []NodeInput    `json:"inferredInputs,omitempty"`
    EffectiveOutput OutputType     `json:"effectiveOutputType,omitempty"`
    EffectiveRoute  []string       `json:"effectiveRouting,omitempty"`
    Warnings        []Diagnostic   `json:"warnings,omitempty"`
    Errors          []Diagnostic   `json:"errors,omitempty"`
}

type Diagnostic struct {
    Code    string `json:"code"`    // E001, W002…
    Message string `json:"message"`
    Hint    string `json:"hint,omitempty"`
}
```

## 四、关键内部类型

```go
// domain/node.go —— 对标 TS 内部 Pin / GraphNode / Edge
type Pin struct {
    ID               string
    Name             string
    Dir              PinDir // In | Out
    Category         string // "materialinput" | "required" | "mask" | ""
    SubCategory      string
    FriendlyName     string
    IsUObjectWrapper bool
    LinkedTo         []PinRef
}

type PinRef struct {
    GraphName string
    PinID     string
}

type GraphNode struct {
    GraphName string
    ExprName  string
    ExprClass string
    IsRoot    bool
    X, Y      int
    NodeGUID  string
    ExprGUID  string
    ExtraBody string
    Pins      []*Pin
    CanRename bool
}

type Edge struct {
    From PinRef
    To   PinRef
}
```

## 五、可测试性设计

1. **GUID 可注入**：`core.GenerateGUID` 通过接口/函数变量注入，测试时用确定性序列。
2. **序列化幂等**：同一输入 + 同一种子 → 同一字节级输出，便于 golden file 对拍。
3. **分层测试**：
   - `parser/` 纯文本 → `NodeInput[]`，单测表驱动。
   - `core/` `GenerateRequest` → `[]*GraphNode`，快照测。
   - `serializer/` `[]*GraphNode` → string，golden 对拍。
   - `app/` 端到端，覆盖 IO 边界。

## 六、第三方库建议（保持极简）

| 用途 | 建议 |
|---|---|
| CLI 参数 | 标准库 `flag` 足够；若子命令多可选 `spf13/cobra` |
| 日志 | 标准库 `log/slog`（Go 1.21+） |
| 测试断言 | 标准库 `testing` + `go-cmp` |
| Release 打包 | `goreleaser`（CI 用） |

**禁止项**：不要引入 YAML、TOML、ORM、数据库、HTTP 框架。此 CLI 是纯文本转换工具。
