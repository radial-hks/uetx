# 核心领域知识（Knowledge Transfer）

> **这是整个设计包最重要的一份文档**。新仓库可以不看任何其他文件，只依据本文件 + `reference/` 下的样本，就能复现一个等价实现。
> 内容全部从 BuilderToolKit 的 TS 实现与真实 UE 样本反推而来。

## 1. 整体目标拓扑

生成器的产物是一组 UE `Begin Object ... End Object` 块，拼接在一起形成"Graph Snippet"。最终长这样：

```
[Root 节点]  MaterialGraphNode_Root_0              ← 材质输出槽（30+ 输入 Pin）
    ├─ Base Color  ──────── LinkedTo ────────┐
    └─ Opacity     ──────── LinkedTo ───┐    │
                                        │    │
[BreakOut 节点，仅 Float4+标量槽时出现]  │    │
MaterialGraphNode_N (MaterialFunctionCall)
    Float4 input  ◀──── Custom.Output     │    │
    A pin  ─────── LinkedTo ──────────────┘    │
                                               │
[Custom 节点]  MaterialGraphNode_0 (Expression=MaterialExpressionCustom)
    Code=..., OutputType=CMOT_FloatN, Inputs(i)=...
    Output pin  ─────── 扇出 ─────────────────┘ + BreakOut.Float4
    inputN pins  ◀──── 来自参数节点

[参数节点 * N]  MaterialGraphNode_1..M
    Scalar / Vector / WorldPosition / Time / TextureCoordinate
```

**关键不变量**：
- Root 固定命名 `MaterialGraphNode_Root_0`。
- 其余节点命名 `MaterialGraphNode_0`, `MaterialGraphNode_1`, ... 顺序递增。
- Custom 节点永远是 `MaterialGraphNode_0`（第一个非 Root 节点）。
- Root 与其他节点的 `Class` 不同：Root 是 `MaterialGraphNode_Root`，其他是 `MaterialGraphNode`。

## 2. Root 节点 Pin 表（常量）

UE 材质 Root 节点的 34 个输入槽，对应 `PinSubCategory` 数字（字符串，非整型字段，但值是数字字符串）。**顺序与编号必须与下表一致**，这是粘贴回 UE 后能正确识别槽位的关键。

```go
// domain/tables.go
var RootPinTable = []struct {
    Name string
    Sub  string // PinSubCategory，注意是字符串
}{
    {"Base Color", "5"},
    {"Metallic", "6"},
    {"Specular", "7"},
    {"Roughness", "8"},
    {"Anisotropy", "9"},
    {"Emissive Color", "0"},
    {"Opacity", "1"},
    {"Opacity Mask", "2"},
    {"Normal", "10"},
    {"Tangent", "11"},
    {"World Position Offset", "12"},
    {"World Displacement", "13"},
    {"Tessellation Multiplier", "14"},
    {"Subsurface Color", "15"},
    {"Custom Data 0", "16"},
    {"Custom Data 1", "17"},
    {"Tree Light Info", "30"},
    {"Ambient Occlusion", "18"},
    {"Refraction", "19"},
    {"Customized UV0", "20"},
    {"Customized UV1", "21"},
    {"Customized UV2", "22"},
    {"Customized UV3", "23"},
    {"Customized UV4", "24"},
    {"Customized UV5", "25"},
    {"Customized UV6", "26"},
    {"Customized UV7", "27"},
    {"Pixel Depth Offset", "28"},
    {"Shading Model", "29"},
    {"Material Attributes", "31"},
}
```

"标量槽"集合（连线到这些槽时，如果上游是 Float4，必须经过 BreakOut 取通道）：

```go
var ScalarSlots = map[string]struct{}{
    "Opacity": {}, "Opacity Mask": {}, "Metallic": {}, "Specular": {},
    "Roughness": {}, "Anisotropy": {}, "Ambient Occlusion": {},
    "Refraction": {}, "Tessellation Multiplier": {}, "Pixel Depth Offset": {},
}
```

## 3. 默认 Routing

```go
func DefaultRouting(t OutputType) []MaterialOutputSlot {
    switch t {
    case CMOTFloat1: return []{"Emissive Color"}
    case CMOTFloat2: return []{"Emissive Color"}
    case CMOTFloat3: return []{"Base Color"}
    case CMOTFloat4: return []{"Base Color", "Opacity"}
    }
}
```

**BreakOut 触发条件**：
```go
needsBreakOut := outputType == CMOTFloat4 &&
    anySlotIn(effectiveRouting, ScalarSlots)
```

## 4. GUID 规则

- 32 个 **大写** 十六进制字符，无连字符。
- `PersistentGuid` 永远是 `00000000000000000000000000000000`（UE 约定）。
- 其他 GUID（`NodeGuid`, `MaterialExpressionGuid`, `PinId`, `ExpressionInputId`, `ExpressionOutputId`）都是新生成的。
- ExprName 里用 `GUID 前 8 位`：如 `MaterialExpressionCustom_33CC8D43`。

Go 实现：
```go
func GenerateGUID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return strings.ToUpper(hex.EncodeToString(b))
}
```

测试时用 `math/rand` 固定种子，或注入 `func() string`。

## 5. 参数节点类型映射

| ParamType | ExprClass | extraBody（缩进 6 空格） | 输出 Pin 数 | canRename |
|---|---|---|---|---|
| `scalar` | `MaterialExpressionScalarParameter` | `DefaultValue=<f6>`，`ParameterName="..."` | 1 | true |
| `vector` | `MaterialExpressionVectorParameter` | `DefaultValue=(R=,G=,B=,A=)`，`ParameterName="..."` | **5**（Output / Output2..5） | true |
| `worldposition` | `MaterialExpressionWorldPosition` | 无 | 1 | false |
| `time` | `MaterialExpressionTime` | 无 | 1 | false |
| `uv` | `MaterialExpressionTextureCoordinate` | 无 | 1 | false |

### Vector 的 5 个输出 Pin（必须全写出）

```
Output      category="mask", subCategory="",        isUObjectWrapper=true
Output2     category="mask", subCategory="red",     isUObjectWrapper=true
Output3     category="mask", subCategory="green",   isUObjectWrapper=false
Output4     category="mask", subCategory="blue",    isUObjectWrapper=false
Output5     category="mask", subCategory="alpha",   isUObjectWrapper=false
```

### 布局坐标（用于 UE 显示，不影响连线）

- Root：`(352, 528)`
- Custom：`(-432, 528)`
- BreakOut：`(-96, 608)`
- 参数节点：`x=-800`，`y = i*150 - len*75 + 528`（i 是 inputs 里的顺序）

## 6. Pin 序列化规则

**完整字段清单**（顺序不可变）：

```
CustomProperties Pin (
  PinId=<32HEX>,
  PinName="<name>",
  [PinFriendlyName=NSLOCTEXT("MaterialGraphNode", "Space", " "),]  ← 仅 Output 类型且有 friendlyName
  [Direction="EGPD_Output",]                                        ← 仅 Out 方向
  PinType.PinCategory="<category>",
  PinType.PinSubCategory="<sub>",
  PinType.PinSubCategoryObject=None,
  PinType.PinSubCategoryMemberReference=(),
  PinType.PinValueType=(),
  PinType.ContainerType=None,
  PinType.bIsReference=False,
  PinType.bIsConst=False,
  PinType.bIsWeakPointer=False,
  PinType.bIsUObjectWrapper=<True|False>,
  [LinkedTo=(<GraphName> <PinId>,<GraphName> <PinId>,),]            ← 非空才写
  PersistentGuid=00000000000000000000000000000000,
  bHidden=False,bNotConnectable=False,
  bDefaultValueIsReadOnly=False,bDefaultValueIsIgnored=False,
  bAdvancedView=False,bOrphanedPin=False,
)
```

**关键注意点**：
- 整段 **一行** 输出，不换行。逗号分隔。
- 前缀缩进：**3 个空格**，然后是 `CustomProperties Pin (`。
- `LinkedTo` 内部格式：`GraphName SPACE PinId,` 每项末尾保留逗号，整个括号末尾也保留逗号。
- `bIsUObjectWrapper` 值类型布尔用 `True/False` 大写。
- 所有其他布尔 `bHidden=False` 等也是 `True/False`。

## 7. 节点序列化骨架

### 非 Root 节点
```
Begin Object Class=/Script/UnrealEd.MaterialGraphNode Name="<graphName>"
   Begin Object Class=/Script/Engine.<exprClass> Name="<exprName>"
   End Object
   Begin Object Name="<exprName>"
      <extraBody>               ← 每行前缀 6 空格
      MaterialExpressionEditorX=<x>
      MaterialExpressionEditorY=<y>
      MaterialExpressionGuid=<32HEX>
      Material=PreviewMaterial'"/Engine/Transient.<materialName>"'
   End Object
   MaterialExpression=<exprClass>'"<exprName>"'
   NodePosX=<x>
   NodePosY=<y>
   [bCanRenameNode=True]        ← 仅 scalar/vector 参数节点
   NodeGuid=<32HEX>
   <Pin 行 * N>
End Object
```

### Root 节点（简化，无内嵌 Expression）
```
Begin Object Class=/Script/UnrealEd.MaterialGraphNode_Root Name="MaterialGraphNode_Root_0"
   Material=PreviewMaterial'"/Engine/Transient.<materialName>"'
   NodePosX=352
   NodePosY=528
   NodeGuid=<32HEX>
   <34 个 Pin 行>
End Object
```

### 节点间分隔符

节点之间用 **单个 CRLF**（不是空行），所有节点拼接后不加 trailing newline。

## 8. Custom 节点的特殊字段

`extraBody` 必须包含（顺序保留）：
```
      Code="<转义后的 HLSL>"
      OutputType=<CMOT_FloatN>
      Inputs(0)=(InputName="...",Input=(Expression=<ExprClass>'"<GraphName>.<ExprName>"'[,Mask=1,MaskR=1,MaskG=1,MaskB=1]))
      Inputs(1)=(...)
      ...
      Desc="Generated by BuilderToolKit"
```

### Code 字符串转义（顺序很重要）
```go
escaped := strings.NewReplacer(
    `\`, `\\`,     // 1. 反斜杠先转
    "\r\n", `\r\n`, // 2. CRLF → 字面量 \r\n
    "\n", `\r\n`,   // 3. 纯 LF 也升级为 \r\n
    `"`, `\"`,      // 4. 引号
).Replace(hlsl)
```
**注意**：UE 的 Code 字段里换行都是 **字面量 `\r\n`**（6 个字符：反斜杠 r 反斜杠 n），不是真实换行。

### Inputs 引用格式
```
Input=(Expression=<ExprClass>'"<GraphName>.<ExprName>"'[,Mask=1,MaskR=1,MaskG=1,MaskB=1])
```
- 必须 **带 GraphName 前缀**：`MaterialGraphNode_3.MaterialExpressionScalarParameter_6F506E71`。
- `vector` + `useRGBMask=true` 时追加 `,Mask=1,MaskR=1,MaskG=1,MaskB=1`。

## 9. BreakOut 节点的特殊字段

引用的 MaterialFunction 路径：
```
/Engine/Functions/Engine_MaterialFunctions02/Utility/BreakOutFloat4Components.BreakOutFloat4Components
```

`extraBody` 必须写 **8 行**：1 个 `FunctionInputs(0)` + 4 个 `FunctionOutputs` + 4 个 `Outputs`：

```
      MaterialFunction=MaterialFunction'"/Engine/Functions/Engine_MaterialFunctions02/Utility/BreakOutFloat4Components.BreakOutFloat4Components"'
      FunctionInputs(0)=(ExpressionInputId=<GUID>,Input=(Expression=MaterialExpressionCustom'"<CustomGraphName>.<CustomExprName>"',InputName="Float4"))
      FunctionOutputs(0)=(ExpressionOutputId=<GUID>,Output=(OutputName="R"))
      FunctionOutputs(1)=(ExpressionOutputId=<GUID>,Output=(OutputName="G"))
      FunctionOutputs(2)=(ExpressionOutputId=<GUID>,Output=(OutputName="B"))
      FunctionOutputs(3)=(ExpressionOutputId=<GUID>,Output=(OutputName="A"))
      Outputs(0)=(OutputName="R")
      Outputs(1)=(OutputName="G")
      Outputs(2)=(OutputName="B")
      Outputs(3)=(OutputName="A")
```

Pin 布局：`Float4 (V4)` 输入 Pin + `R/G/B/A` 四个输出 Pin。

## 10. 结构等价比对规则

golden 对拍时 **不能直接 `bytes.Equal`**（GUID 不稳定）。定义等价关系 ≡：

两份 T3D `a ≡ b` 当且仅当：

1. 节点数量一致，节点类型按拓扑顺序一致（Root → Custom → [BreakOut?] → Params*）。
2. 每个节点的 `ExprClass` 一致。
3. 每个节点的 Pin 集合（按 `PinName + Direction`）一致。
4. 连线图同构：把 `LinkedTo` 与 `Inputs(i)` 视为有向边，两图节点和边可一一映射（忽略 GUID、忽略节点/Pin 名的具体哈希前缀）。
5. Custom 节点的 `Code`（反转义后）一致。
6. Custom 节点的 `OutputType`、`Inputs(i).InputName`、`Inputs(i).Mask` 一致。
7. Root 的 34 个 Pin 的 `Name` 和 `SubCategory` 与 [§2](#2-root-节点-pin-表常量) 完全一致。

实现建议：写一个 `internal/testutil/t3dparse.go` 做极简反解析，构建结构体后做字段对比。**不要**用正则做对比，会很脆弱。

## 11. 模板解析规则（parser）

### 仅在第一个 `/* ... */` 注释块内扫描
避免在代码体里误匹配。

### Pin 正则（大小写不敏感，`g` 全局）
```
Pin\s+\d+\s+Name:\s*\[([^\]]+)\]\s*\|\s*Type suggestion:\s*([^\n|(]+)(?:\s*\(Default:\s*([^)]+)\))?
```
- Group1: Pin 名（去首尾空格）
- Group2: Type suggestion 原文（下游做类型推断）
- Group3: Default 原文（可选）

### Type 推断（顺序很重要，先到先得）
```
if contains("world") && contains("position") → worldposition
else if contains("time")                      → time
else if contains("texture")||"coord"||"uv"    → uv
else if contains("vector")||"color"           → vector
else                                          → scalar  // 默认
```

### OutputType 正则
```
Output Type\s*\(输出类型\):\s*\[(.*?)\]
```
归一化：
1. `Float\s+(\d)` → `Float$1`（去掉 "Float 4" 中间空格）
2. 全部 `\s+` → `_`（"CMOT Float4" → "CMOT_Float4"）
3. 校验是否属于 `CMOT_Float1..4`

## 12. 边界情况与必须处理的 Warning

| 场景 | 处理 | 代码 |
|---|---|---|
| HLSL 无注释块 | 只用 CLI 配置，否则报 E002 | E002 |
| 解析到 0 Pin 但 CLI 也没传 | 生成无输入的 Custom 节点，合法 | W002 |
| Pin 名与代码体变量不一致 | 警告不阻塞 | W003 |
| `useRGBMask=true` 但非 vector | 忽略该字段 | W004 |
| routing 含 `ScalarSlots` 但 outputType 非 Float4 | 直接连，不生成 BreakOut；给警告 | W005 |
| `materialName` 为空 | 回退 `M_CustomNode` | - |
| `materialName` 含空格 / 非法字符 | 报错 E110 |
| HLSL 末尾无 `return` | 警告 | W001 |
| GUID 冲突（概率 ~0） | 运行时 panic，记录 bug |

## 13. 参考：TS 源码原样保留（oracle）

为方便在 Go 实现里做逐函数对照，建议把以下 TS 文件作为"参考实现"拷贝到 `reference/`（已在本设计包的 `reference/` 目录里）：

- `T3DGenerator.ts`：`buildT3D` 全流程、Pin/Node 序列化。
- `UENodeGenerator.tsx` 中的 `parseTemplate` 函数。
- `M_WaterLevel_webeditor.txt`：模板输入样本。
- `M_WaterLevel_unrealeditor.txt`：UE 真机导出样本（对拍真相）。

## 14. 可复现性清单（实现完成判据）

- [ ] Root 节点 34 Pin 字节级与 `M_WaterLevel_unrealeditor.txt` 的 Root 部分等价（Name/Sub）。
- [ ] 7 个 input 的 `M_WaterLevel` 样本通过 §10 等价比对。
- [ ] 切换 routing 触发/不触发 BreakOut 两种路径都被测试覆盖。
- [ ] Vector 参数带 RGB Mask 与不带的 `Inputs(i)` 行都被测试覆盖。
- [ ] `seed` 固定时输出字节一致（golden 稳定）。
- [ ] CRLF 字节检查通过（`0x0D 0x0A` 换行）。
- [ ] Windows amd64 与 macOS arm64 都能交叉编译出单 exe。
