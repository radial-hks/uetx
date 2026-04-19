# reference/

本目录是 **Go 版 CLI 的参考实现 / 对拍基线**，不作为编译源码。迁移到新仓库时整体拷贝过去。

## 文件

| 文件 | 角色 | 用途 |
|---|---|---|
| `T3DGenerator.ts` | Oracle（权威参考实现） | 对照 `buildT3D` 各阶段行为，尤其是 Pin/Node 序列化细节 |
| `UENodeGenerator.tsx` | Oracle（模板解析） | 只看其中的 `parseTemplate` 函数 |
| `M_WaterLevel_webeditor.txt` | 输入样本 | 含 7 个 inputs 的 HLSL 模板（Universal Template 格式） |
| `M_WaterLevel_unrealeditor.txt` | Golden 输出样本 | UE 真机导出的 T3D，结构等价比对的真相源 |

## 使用方式

1. **对拍**：Go 实现生成的 T3D 经反解析后，与 `M_WaterLevel_unrealeditor.txt` 走 `CORE_KNOWLEDGE.md §10` 的结构等价关系比对。
2. **行为对照**：实现某个函数前，先读对应的 TS 函数，再按 `CORE_KNOWLEDGE.md` 的规则落地。**不要盲目翻译 TS**，`CORE_KNOWLEDGE.md` 才是规格。
3. **Web 产物对比**：`M_WaterLevel_webeditor.txt` 是当前 Web 工具生成的结果，保留用于展示 Web 版与 UE 真机版的差距（设计文档 `docs/UENodeGenerator/UENodeGenerator_Design.md` 分析过）。

## 不要做的事

- 不要直接用 `diff` 做对拍（GUID 不稳定）。
- 不要把这些 TS 文件当新仓库的源码去编译；它们是文档资产。
- 不要在新仓库里修改本目录内容；要更新规则请修改 `CORE_KNOWLEDGE.md`。
