# uetx CLI 设计包

> 本目录是 **独立 Go CLI 项目** 的完整设计方案与知识转移包。
> 目的：在新仓库从零开始开发时，所有业务规则、格式细节、边界条件都能不依赖 BuilderToolKit Web 仓库直接复现。

## 目录

| 文件 | 作用 |
|---|---|
| [README.md](./README.md) | 总览、技术决策、范围界定 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 目录结构、模块职责、依赖方向 |
| [COMMANDS.md](./COMMANDS.md) | CLI 命令设计、参数、退出码 |
| [IO_PROTOCOL.md](./IO_PROTOCOL.md) | JSON Schema、错误码、stdin/stdout 约定 |
| [CORE_KNOWLEDGE.md](./CORE_KNOWLEDGE.md) | 领域核心知识：Pin 表、T3D 序列化规则、边界情况 |
| [ROADMAP.md](./ROADMAP.md) | 分阶段实施计划、验收基线 |
| [reference/](./reference/) | 从 BuilderToolKit 抽取的权威样本与源码快照 |

## 一、项目定位

**名称**：`uetx` — **Unreal Engine Text eXchange**

**一句话定义**：
一个覆盖 Unreal 文本化资产（T3D / Material / Blueprint / DataTable 等）**双向转换**的 Go CLI 与 SDK。核心能力是"文本 ⇄ 结构化 IR ⇄ 文本"，让 Skill、脚本、CI 都能以稳定契约消费 UE 资产。

**v1.0 范围（MVP）**：
- Material Custom Node：HLSL 模板 → T3D Graph Snippet（本设计包主要聚焦这一条）。

**后续扩展方向（已预留架构空间，不在 MVP 实现）**：
- `uetx material parse` — T3D → 结构化 IR（反向）
- `uetx blueprint generate` — 蓝图文本生成
- `uetx blueprint parse` — 蓝图 T3D → IR
- `uetx t3d diff` / `uetx t3d validate` — 通用 T3D 诊断
- `uetx datatable` / `uetx level` 等按需扩展

**子命令空间规划**：
```
uetx material <generate|parse|validate|diff>
uetx blueprint <generate|parse|validate|diff>
uetx t3d       <validate|diff|format>
uetx version | help
```

**使用形态**：
1. **Skill 入口**：Claude/Copilot Skill 通过 stdin/stdout 调用 `uetx material generate`。
2. **本地 CLI**：美术/TA 手动命令行调用，输出文件或写入剪贴板。
3. **批处理**：CI 脚本批量转换素材目录。
4. **SDK**：`pkg/uetx/...` 供其他 Go 程序直接 import。

**非目标（明确不做）**：
- 不做 HLSL 语法校验 / 编译。
- 不做 Material Editor UI。
- 不直接调用 Gemini / 任何 LLM（Prompt 生成属于上游职责）。
- 不做二进制 `.uasset` 解析（只处理 UE 文本导出格式）。
- 不做增量 diff / patch 现有 `.uasset`。

## 二、技术选型：Go

### 决策

**正式交付：Go 1.22+，单文件 exe，主目标 Windows amd64，扩展 macOS arm64/amd64。**

### 理由

| 维度 | Go | Bun+TS | Rust |
|---|---|---|---|
| 分发（Windows 单 exe） | ✅ 原生静态编译 | ⚠️ 需打包运行时或用户装 Bun | ✅ |
| 跨平台（Win/mac） | ✅ `GOOS/GOARCH` 一条命令 | ⚠️ Bun mac/win 覆盖度弱于 node | ✅ |
| 开发/维护门槛 | 低 | 最低（可复用现有 TS） | 高 |
| 字符串/文本工具 | 强，`text/template`、`bufio` 合用 | 最强（已有实现） | 中 |
| Skill 集成（stdin/stdout + JSON） | ✅ `encoding/json` 一等公民 | ✅ | ✅ |
| 本任务 CPU/内存收益 | 足够 | 足够 | 过度 |
| 二进制体积 | ~5–10 MB | 需打包 ~50 MB+ | ~1–3 MB |
| 启动延迟（Skill 每次调用） | <20 ms | 100–300 ms | <20 ms |

**关键权衡**：
- **不选 Bun**：交付边界模糊（运行时 or 打包？），Windows-first 长期分发不稳。
- **不选 Rust**：业务是纯文本 IR 构建，Rust 带来的所有权/并发优势在此问题上收益极低，反而抬高维护门槛。
- **选 Go**：最匹配"零依赖单 exe + 文本处理 + JSON 协议 + 跨平台"这组约束。

### 原型期过渡

在 Go 版落地前，允许在 BuilderToolKit 仓库内把 Web 的 `parseTemplate` 与 `buildT3D` 继续作为"规则权威"使用。Go 版本落地后，**BuilderToolKit 的 TS 实现将成为参考实现（oracle），用于回归对拍**，不再是唯一事实源。

## 三、范围与边界

```
输入                           处理                           输出
─────────────────────────────────────────────────────────────────
HLSL 文本（含 Universal     ┌───────────────┐   T3D Graph Snippet
Template 注释块）      ───▶ │ uetx core   │ ─▶ （CRLF，含 Root
                            │               │    + Custom + Params
JSON 配置（可选覆盖         │ parse → IR    │    + BreakOut）
inputs/routing/            │ → serialize   │
materialName）        ───▶ └───────────────┘ ─▶ JSON 诊断
                                              （inferred inputs,
                                               warnings, errors）
```

详见 [IO_PROTOCOL.md](./IO_PROTOCOL.md)。

## 四、黄金样本（回归基线）

所有实现必须能将 `reference/M_WaterLevel_webeditor.txt`（HLSL 源）转换成与 `reference/M_WaterLevel_unrealeditor.txt`（UE 真实导出）**结构等价**的 T3D。

"结构等价"定义见 [CORE_KNOWLEDGE.md §10](./CORE_KNOWLEDGE.md#10-结构等价比对规则)。

## 五、阅读顺序建议

1. 本文件（定位 / 决策）
2. [CORE_KNOWLEDGE.md](./CORE_KNOWLEDGE.md) ← **最重要**，领域规则都在这里
3. [IO_PROTOCOL.md](./IO_PROTOCOL.md) ← 定契约
4. [ARCHITECTURE.md](./ARCHITECTURE.md) ← 定目录
5. [COMMANDS.md](./COMMANDS.md) ← 定 UX
6. [ROADMAP.md](./ROADMAP.md) ← 定节奏
