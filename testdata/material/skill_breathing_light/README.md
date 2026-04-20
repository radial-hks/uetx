# 自然呼吸灯 Skill 测试文档

## 1. 测试目标

- 可执行文件: `scripts/uetx.exe`
- 版本: `uetx 1.0.0 (b376231)`
- 测试目标: 使用 Skill 工作流生成一个“自定义颜色切换的自然呼吸灯”材质图，并保留 HLSL 与 T3D 产物。

## 2. 效果说明

本次生成的是一个双色平滑切换的呼吸灯效果，重点不是机械性明灭，而是更接近真实呼吸节奏：

- 呼吸亮度使用平滑正弦作为基础节奏。
- 吸气与呼气使用不同的曲线塑形，避免生硬对称。
- 峰值附近增加短暂停留，减少“打点感”。
- 颜色切换与亮度呼吸不同速，并在亮度峰值附近轻微偏向第二颜色，避免颜色死板来回摆动。

本次路由为：`Base Color` + `Emissive Color`。

## 3. 产物清单

- HLSL 模板: `breathing_color_switch.hlsl`
- 生成结果: `breathing_color_switch.t3d`
- inspect 输出: `breathing_color_switch.inspect.json`
- validate 输出: `breathing_color_switch.validate.json`
- generate 输出: `breathing_color_switch.generate.json`

说明：JSON 文件是本次命令执行过程中保留的原始辅助产物，用于回看 CLI 响应。

## 4. 执行命令

### 4.1 版本检查

```powershell
scripts\uetx.exe version
```

### 4.2 inspect

```powershell
scripts\uetx.exe inspect -i testdata\material\skill_breathing_light\breathing_color_switch.hlsl --json
```

### 4.3 validate

```powershell
scripts\uetx.exe validate -i testdata\material\skill_breathing_light\breathing_color_switch.hlsl --json
```

### 4.4 generate T3D

```powershell
scripts\uetx.exe generate -i testdata\material\skill_breathing_light\breathing_color_switch.hlsl -o testdata\material\skill_breathing_light\breathing_color_switch.t3d -m M_BreathingColorLight -r "Base Color" -r "Emissive Color" --seed 20260420
```

### 4.5 generate JSON 汇总

```powershell
scripts\uetx.exe generate -i testdata\material\skill_breathing_light\breathing_color_switch.hlsl -m M_BreathingColorLight -r "Base Color" -r "Emissive Color" --seed 20260420 --json
```

## 5. 测试结果

### 5.1 inspect

- 结果: 成功
- `ok = true`
- 推断输入数量: `10`
- 推断输出类型: `CMOT_Float3`
- 默认路由: `Base Color`

### 5.2 validate

- 结果: 成功
- `ok = true`
- 无错误
- 无警告

### 5.3 generate

- 结果: 成功
- 材质名: `M_BreathingColorLight`
- 生效路由: `Base Color`, `Emissive Color`
- `nodeCount = 12`
- `edgeCount = 12`
- `hasBreakOut = false`
- 无错误
- 无警告

## 6. 参数建议

当前模板暴露的参数如下：

- `ColorA`: 第一段颜色
- `ColorB`: 第二段颜色
- `MinIntensity`: 最低亮度
- `MaxIntensity`: 最高亮度
- `BreathSpeed`: 呼吸速度
- `ColorShiftSpeed`: 颜色切换速度
- `PhaseOffset`: 相位偏移，可用于做多灯错峰
- `BreathSoftness`: 呼吸塑形柔和度
- `ColorBlendSoftness`: 颜色混合柔和度

推荐起步值：

- `BreathSpeed = 0.32`
- `ColorShiftSpeed = 0.18`
- `MinIntensity = 0.12`
- `MaxIntensity = 3.50`
- `BreathSoftness = 1.40`
- `ColorBlendSoftness = 1.15`

如果要更偏“安静设备呼吸灯”，可以进一步降低 `BreathSpeed` 与 `ColorShiftSpeed`。

## 7. Skill 集成中的改进点

以下问题都来自这次对 `scripts/uetx.exe` 的实际调用过程，目的是让后续 Skill 更易用。

### 改进点 1: `--json` 与 `-o` 不能同时满足“写文件 + 返回机器可读摘要”

当前 `generate --json` 会直接把 JSON 打到 stdout，并跳过 `-o` 写文件流程。结果是 Skill 如果既想保留 `.t3d` 文件，又想拿到 `stats/materialName/effectiveRouting` 等机器可读数据，就必须执行两次 `generate`。

建议：

- 增加 `--json-out <path>`。
- 或允许 `--json` 与 `-o` 同时生效：stdout 输出 JSON，`-o` 正常写 T3D。

### 改进点 2: Windows PowerShell 下 JSON 重定向容易变成 UTF-16LE

这次为了保留 JSON 响应，使用了 PowerShell 重定向保存 stdout。结果产物文件是 UTF-16LE BOM，这对后续脚本处理、diff 查看和跨工具消费都不够友好。

建议：

- 增加 `--json-out <path>`，并明确固定写 UTF-8。
- 同理可考虑 `--stderr-out <path>` 或统一 artifact 输出目录。

### 改进点 3: `inspect` / `validate` 不能完整复现最终 `generate` 的上下文

当前 `inspect` 和 `validate` 没有和 `generate` 对齐的全部覆盖参数组合，例如最终路由、显式输入覆盖、材质名等上下文不容易在预检阶段完全重现。

建议：

- 让 `inspect` / `validate` 也支持 `-m`、`-r`、`--input`、`-t`。
- 这样 Skill 可以先做“最终配置的预检”，再做一次真正生成。

### 改进点 4: 缺少面向 Skill 的统一产物输出模式

这次流程里我们分别保留了 HLSL、T3D、inspect JSON、validate JSON、generate JSON。对 Skill 来说，这是一组天然相关的产物，但当前要靠外部脚本逐个保存。

建议：

- 增加 `--artifact-dir <dir>`。
- 一次运行后在目录中输出：`request.json`、`inspect.json`、`generate.json`、`output.t3d`。
- 同时可选地输出 `effective-config.json`，方便溯源和复测。

## 8. 结论

本次 Skill 流程已经成功落地，且生成结果满足目标：

- 效果是自然的双色呼吸灯，而不是机械闪烁。
- HLSL 和 T3D 文件均已保留。
- 生成图结构稳定，可复现，使用固定 `--seed 20260420` 可得到一致结果。

如果后续要把这个流程进一步产品化，优先建议处理 `--json-out` 和 `--artifact-dir`，这两项对 Skill 集成收益最高。