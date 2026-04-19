# 实施路线

## 原则

- **先跑通 golden 对拍，再做 CLI**。核心价值是 T3D 正确性，命令面只是外壳。
- **每个阶段都要可独立验证**，不做未经测试的多阶段连写。
- **TS 实现作为 oracle**：同一输入在 TS 与 Go 下产出应通过 §10 等价比对。

## 阶段 0：项目骨架（半天）

- 初始化 `go mod`（建议 module 路径 `github.com/<you>/uetx`）
- 建立 `cmd/`、`internal/`、`testdata/` 目录
- 拷贝本设计包到 `docs/`
- 拷贝 `reference/M_WaterLevel_*.txt` 到 `testdata/golden/`
- 写最小 `main.go`：`uetx version` 能跑

**验收**：`go build ./... && ./uetx version` 成功。

## 阶段 1：Domain + Tables（1 天）

- `domain/types.go`：枚举 + 核心结构体
- `domain/tables.go`：`RootPinTable`、`ScalarSlots`、`DefaultRouting`
- `core/guid.go`：`GenerateGUID` + 可注入种子

**验收**：`go test ./domain/...` 覆盖 Pin 表长度（34）、sub 数字字符串正确性。

## 阶段 2：Parser（1 天）

- `parser/template.go`：实现 §11 两组正则
- `parser/typeinfer.go`：类型推断分支
- 表驱动测试：覆盖 `reference/` 两个 HLSL 模板 + 边界情况

**验收**：解析 `M_WaterLevel.hlsl` 得到 7 个正确类型的 `NodeInput` + `CMOT_Float4`。

## 阶段 3：Core IR Builder（2 天）

按 TS `buildT3D` 逐步翻译，拆成几个子函数：
- `buildRoot()` → GraphNode（Root，34 Pin）
- `buildCustom(inputs, hlsl, outputType)` → GraphNode
- `buildParams(inputs)` → []GraphNode
- `buildBreakOut(custom)` → *GraphNode（可能 nil）
- `buildEdges(routing, breakOutPresent)` → []Edge
- `applyEdges(edges, nodes)`：双向 LinkedTo 回填

**验收**：对 `M_WaterLevel.hlsl` 构建的 IR 节点数 = 9（Root + Custom + BreakOut + 7 params），edges 数正确。

## 阶段 4：Serializer（1.5 天）

- `serializer/escape.go`：Code 字段转义（§8）
- `serializer/t3d.go`：Pin 行、Node 体、Root 体
- CRLF 输出 + 3 空格 / 6 空格缩进规则

**验收**：对 `M_WaterLevel.hlsl` 序列化结果经反解析后与 `M_WaterLevel_unrealeditor.txt` 通过 §10 等价比对。

## 阶段 5：App Layer（1 天）

- `app/generate.go`：配置合并（默认 < 模板 < JSON 文件 < CLI flags）
- `app/inspect.go`、`app/validate.go`
- `app/diagnostics.go`：warning/error 收集

**验收**：`GenerateRequest` → `GenerateResponse` 端到端测试通过。

## 阶段 6：CLI（1 天）

- `cmd/uetx/main.go`：子命令分发
- stdin/stdout 模式、`--json`、`--stdin-json`
- 退出码规范
- 剪贴板适配（Windows `clip.exe` / macOS `pbcopy`）

**验收**：
- `cat shader.hlsl | ./uetx generate -o out.t3d` 成功
- `echo '{...}' | ./uetx generate --stdin-json --json` 返回合法 JSON

## 阶段 7：发布（0.5 天）

- GitHub Actions：`ci.yml`（test + vet）、`release.yml`（goreleaser）
- 产物：`uetx_windows_amd64.exe`、`uetx_darwin_arm64`、`uetx_darwin_amd64`
- Skill 接入文档（如何在 SKILL.md 里调用）

**验收**：tag 推送后 GitHub Release 自动产出三个平台的二进制。

## 里程碑总览

| 阶段 | 关键产物 | 依赖 |
|---|---|---|
| 0 | 项目骨架 | - |
| 1 | domain + tables | 0 |
| 2 | parser | 1 |
| 3 | IR builder | 1 |
| 4 | serializer | 3 |
| 5 | app layer | 2,4 |
| 6 | CLI | 5 |
| 7 | 发布 | 6 |

## 风险与缓解

| 风险 | 缓解 |
|---|---|
| UE 版本差异导致 Pin 表偏移 | 把 `RootPinTable` 做成可扩展映射（未来按 UE 版本分表） |
| GUID 随机性破坏 golden 稳定 | `seed` 固定 + 测试中显式注入 |
| TS 与 Go 正则引擎 Unicode 行为差异 | 解析阶段规范化为 LF；测试覆盖中文注释样本 |
| Windows CRLF 与 Go stdout 默认行为 | 显式 `\r\n`，不依赖平台换行 |
| 剪贴板跨平台坑 | 调用系统命令而非绑定库；失败降级为 warning |

## 完成后续建议（不在 MVP 范围内）

- `--watch` 模式：文件变更自动重跑
- `template` 子命令：`uetx template list|show <name>` 输出内置模板
- `--from-clipboard`：从剪贴板读取 HLSL
- LSP 化：把解析/校验做成语言服务器，供 VS Code 使用
- 反向工具：T3D → HLSL 模板（还原）
