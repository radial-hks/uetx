/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): [自定义双色平滑切换的自然呼吸灯效果]
 * - Output Type (输出类型): [CMOT Float 3]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [Time]                | Type suggestion: Time
 * Pin 1 Name: [ColorA]              | Type suggestion: Vector 3 (Default: 1.0, 0.45, 0.20)
 * Pin 2 Name: [ColorB]              | Type suggestion: Vector 3 (Default: 0.20, 0.85, 1.0)
 * Pin 3 Name: [MinIntensity]        | Type suggestion: Scalar (Default: 0.12)
 * Pin 4 Name: [MaxIntensity]        | Type suggestion: Scalar (Default: 3.50)
 * Pin 5 Name: [BreathSpeed]         | Type suggestion: Scalar (Default: 0.32)
 * Pin 6 Name: [ColorShiftSpeed]     | Type suggestion: Scalar (Default: 0.18)
 * Pin 7 Name: [PhaseOffset]         | Type suggestion: Scalar (Default: 0.0)
 * Pin 8 Name: [BreathSoftness]      | Type suggestion: Scalar (Default: 1.40)
 * Pin 9 Name: [ColorBlendSoftness]  | Type suggestion: Scalar (Default: 1.15)
 * =================================================================================
 */

// --- [CODE BODY START] ---

static const float TWO_PI = 6.28318530718;
static const float EPSILON = 0.0001;

float safeBreathSoftness = max(BreathSoftness, EPSILON);
float safeColorSoftness = max(ColorBlendSoftness, EPSILON);
float intensityLow = min(MinIntensity, MaxIntensity);
float intensityHigh = max(MinIntensity, MaxIntensity);

float breathPhase = frac(Time * BreathSpeed + PhaseOffset);
float breathWave = 0.5 - 0.5 * cos(breathPhase * TWO_PI);

float inhaleCurve = pow(saturate(breathWave), safeBreathSoftness);
float exhaleCurve = 1.0 - pow(saturate(1.0 - breathWave), safeBreathSoftness * 1.35);
float breathMask = lerp(inhaleCurve, exhaleCurve, breathWave);
breathMask = smoothstep(0.0, 1.0, breathMask);

float peakHold = smoothstep(0.72, 0.98, breathWave);
float releaseSoftener = 1.0 - 0.08 * smoothstep(0.98, 1.0, breathWave);
breathMask = saturate(lerp(breathMask, 1.0, peakHold * 0.18) * releaseSoftener);

float intensity = lerp(intensityLow, intensityHigh, breathMask);

float colorPhase = frac(Time * ColorShiftSpeed + PhaseOffset * 0.37 + breathMask * 0.08);
float colorWave = 0.5 - 0.5 * cos(colorPhase * TWO_PI);
float signedColorWave = colorWave * 2.0 - 1.0;
float colorBlend = sign(signedColorWave) * pow(abs(signedColorWave), safeColorSoftness);
colorBlend = smoothstep(0.0, 1.0, colorBlend * 0.5 + 0.5);

float3 blendedColor = lerp(ColorA, ColorB, colorBlend);
float3 breathingColor = lerp(blendedColor, ColorB, peakHold * 0.10);
float3 result = breathingColor * intensity;

return result;

// --- [CODE BODY END] ---