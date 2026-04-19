# IO 协议

本文件定义 uetx CLI 与调用方（Skill、脚本、其他工具）之间的数据契约。**任何实现变更都应先改本文件。**

## 一、JSON Schema

### 1.1 `GenerateRequest`

```json
{
  "hlsl": "string (必填, UTF-8, 可含 CRLF 或 LF)",
  "materialName": "string (可选, 默认 M_CustomNode)",
  "outputType": "CMOT_Float1 | CMOT_Float2 | CMOT_Float3 | CMOT_Float4 (可选)",
  "inputs": [
    {
      "name": "string",
      "type": "scalar | vector | time | uv | worldposition",
      "defaultValue": "string (可选)",
      "useRGBMask": "boolean (可选, 仅 vector 有意义)"
    }
  ],
  "routing": ["Base Color", "Opacity", "..."],
  "seed": "int64 (可选, 固定 GUID 序列)"
}
```

**字段可选性规则**：
- 只传 `hlsl` → 全走模板解析 + 默认值（最常用）。
- 传 `inputs` → 覆盖解析结果（解析被禁用）。
- 传 `outputType` → 覆盖解析结果。
- 传 `routing` → 覆盖默认路由。

### 1.2 `GenerateResponse`

```json
{
  "ok": true,
  "t3d": "Begin Object Class=...\r\n...",
  "inferredInputs": [ { "name": "...", "type": "...", "defaultValue": "..." } ],
  "effectiveOutputType": "CMOT_Float4",
  "effectiveRouting": ["Base Color", "Opacity"],
  "materialName": "M_CustomNode",
  "stats": {
    "nodeCount": 9,
    "edgeCount": 9,
    "hasBreakOut": true
  },
  "warnings": [ { "code": "W001", "message": "...", "hint": "..." } ],
  "errors": []
}
```

失败时：
```json
{
  "ok": false,
  "errors": [ { "code": "E101", "message": "Output Type 无法识别: 'Float 5'", "hint": "使用 [CMOT Float 1..4]" } ]
}
```

## 二、stdin / stdout 约定

### `generate` 的三种调用模式

| 模式 | stdin | flags | stdout |
|---|---|---|---|
| **纯文本 T3D** | HLSL | 无 `--json` | T3D（CRLF） |
| **JSON 响应** | HLSL | `--json` | 单行 JSON（UTF-8，末尾换行） |
| **JSON 请求+JSON 响应** | JSON (`GenerateRequest`) | `--stdin-json --json` | 单行 JSON |

**Skill 推荐组合**：`--stdin-json --json`。调用方拼 `GenerateRequest`，解析 `GenerateResponse`，全程结构化，不依赖参数字符串。

示例：
```bash
echo '{"hlsl":"...","materialName":"M_Water"}' \
  | uetx generate --stdin-json --json
```

## 三、错误码表

### 解析类（E0xx）
| Code | 含义 |
|---|---|
| E001 | HLSL 为空 |
| E002 | 未找到模板注释块 `/* ... */` |
| E010 | Pin 行格式错误（缺少 `[...]` 或 `|`） |
| E011 | 重复 Pin 名称 |
| E012 | Pin 类型推断失败 |
| E020 | Output Type 无法识别 |

### 配置类（E1xx）
| Code | 含义 |
|---|---|
| E100 | JSON 配置解析失败 |
| E101 | `outputType` 非法值 |
| E102 | `routing` 含未知 slot |
| E103 | `inputs[i].type` 非法 |
| E110 | `materialName` 含非法字符（允许 `[A-Za-z0-9_]`） |

### 构建类（E2xx）
| Code | 含义 |
|---|---|
| E200 | IR 构建内部错误 |
| E201 | routing 要求 BreakOut 但 outputType 非 Float4 |
| E210 | Edge 断裂（from/to 引用了不存在的 Pin） |

### 警告类（W0xx）
| Code | 含义 |
|---|---|
| W001 | HLSL 未声明 return（生成物可能无效） |
| W002 | 模板块解析到 0 个 Pin |
| W003 | 某 Pin 在代码体中未出现 |
| W004 | `useRGBMask=true` 但 type 非 vector |
| W005 | routing 为空且模板未指定，回退 defaultRouting |
| W006 | 剪贴板复制失败（仅 `--clipboard`） |

## 四、换行与编码

- **输入**：接受 LF 或 CRLF，内部规范化为 LF 处理。
- **输出 T3D**：**必须 CRLF**。UE 原生 `.t3d` 是 CRLF，这是"可粘贴回 UE"的必要条件之一。
- **输出 JSON**：LF，紧凑格式（无缩进），末尾一个换行。
- **编码**：全链路 UTF-8，无 BOM。

## 五、幂等性

给定同一 `GenerateRequest` 与同一 `seed`，输出必须字节级一致。这是 golden test 的基础。

## 六、版本策略

- Semver：`MAJOR.MINOR.PATCH`。
- **Breaking 定义**：JSON 字段重命名/删除、错误码含义变更、T3D 字节格式对等性丢失。
- 响应中建议加 `"schemaVersion": "1.0"` 字段（放顶层），Skill 方据此兼容。
