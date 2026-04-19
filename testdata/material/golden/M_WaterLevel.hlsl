/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): 基于世界空间Z轴高度区分水上和水下状态，并分别赋予自定义颜色（包含透明度）。输出包含Alpha的四维颜色值。
 * - Output Type (输出类型): [CMOT Float 4]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [WorldPosition]    | Type suggestion: Vector 3 (推荐连接: Absolute World Position)
 * Pin 1 Name: [WaterLevel]       | Type suggestion: Scalar (推荐: 暴露为材质参数)
 * Pin 2 Name: [Feather]          | Type suggestion: Scalar (推荐: 暴露为材质参数，例如 10.0)
 * Pin 3 Name: [AboveWaterColor]  | Type suggestion: Vector 3 (水上基础颜色 RGB)
 * Pin 4 Name: [AboveWaterAlpha]  | Type suggestion: Scalar   (水上透明度 Alpha)
 * Pin 5 Name: [UnderwaterColor]  | Type suggestion: Vector 3 (水下基础颜色 RGB)
 * Pin 6 Name: [UnderwaterAlpha]  | Type suggestion: Scalar   (水下透明度 Alpha)
 * =================================================================================
 */

// --- [CODE BODY START] ---

// 1. 将输入的 RGB 和 Alpha 组装成安全的 float4 颜色
// 这可以彻底避免 float3 无法隐式转换为 float4 的报错
float4 colorAbove = float4(AboveWaterColor, AboveWaterAlpha);
float4 colorUnder = float4(UnderwaterColor, UnderwaterAlpha);

// 2. 安全处理：防止羽化值为0导致除以0的报错
float safeFeather = max(Feather, 0.0001);

// 3. 计算高度差 (水面高度 - 像素当前世界高度)
// 当像素位于水下时 (WorldPosition.z < WaterLevel)，depthDiff 为正数
// 当像素位于水上时 (WorldPosition.z > WaterLevel)，depthDiff 为负数
float depthDiff = WaterLevel - WorldPosition.z;

// 4. 计算浸没遮罩并限制在 0~1 之间
float mask = saturate(depthDiff / safeFeather);

// 5. 根据遮罩混合水上和水下颜色 (包括 Alpha 通道)
float4 finalColor = lerp(colorAbove, colorUnder, mask);

// Return
return finalColor;

// --- [CODE BODY END] ---
