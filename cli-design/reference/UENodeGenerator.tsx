import React, { useState, useEffect } from 'react';
import { IconCopy, IconCheck, IconCode, IconUnreal, IconSparkles, IconArrowRight, IconX } from '../Icons';
import { buildT3D, generateGuid, ParamType, OutputType, NodeInput, MaterialOutputSlot } from './ue/T3DGenerator';

// --- Templates ---

const SYSTEM_PROMPT = `
# Role
You are an expert Unreal Engine Technical Artist and Shader Programmer. Your task is to generate optimized HLSL code for a Material "Custom Node".

# Goal
Generate an HLSL shader snippet based on the user's request, strictly following the "Universal Template" format.

# ⚠️ CRITICAL FORMATTING RULES (Must Follow)
1. **Pin Naming Syntax**: 
   - Every input variable name in the "INPUTS" section MUST be enclosed in square brackets \`[]\`.
   - Example: Use \`[Time]\`, \`[UV]\`. Do NOT use \`Time\`, \`UV\` without brackets.
   - The variable name inside the code body must match the Pin Name exactly (without brackets).

2. **Output Type Syntax**:
   - The value for "Output Type" MUST be enclosed in square brackets \`[]\`.
   - Correct: \`[CMOT Float 1]\`, \`[CMOT Float 3]\`, \`[CMOT Float 4]\`
   - Incorrect: \`CMOT Float 1\` (Missing brackets is FORBIDDEN).

3. **Type Suggestion Precision**: 
   - You must provide accurate Unreal Engine Material Editor pin types in the \`Type suggestion\` field. 
   - Use ONLY the following standard types:
     - \`float\` -> **Scalar**
     - \`float2\` -> **Vector 2** or **TextureCoordinate**
     - \`float3\` -> **Vector 3**
     - \`float4\` -> **Vector 4**
     - \`Texture2D\` -> **Texture Object**

4. **Language**: 
   - The \`Description\` and comments within the template must be in **Chinese (Simplified)**.
   - The code logic itself must be standard HLSL.

# Universal Template
You must output the result strictly inside this block:

/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): [Brief description in Chinese]
 * - Output Type (输出类型): [Select one: [CMOT Float 1] | [CMOT Float 3] | [CMOT Float 4]]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [InputName1]     | Type suggestion: [Scalar/Vector 3/etc.] (Default: X)
 * Pin 1 Name: [InputName2]     | Type suggestion: [Scalar/Vector 3/etc.] (Default: X)
 * [Add more pins if needed...]
 * =================================================================================
 */

// --- [CODE BODY START] ---

// 1. Logic Step...
[Your HLSL Code Here]

// ...

// Return
return [Result];

// --- [CODE BODY END] ---

# User Request
Please implement the following effect:
`;

const TEMPLATES = {
  raindrop: {
    name: "Raindrop Ripple",
    code: `/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): [基于网格的程序化雨滴涟漪效果。返回RGB高度可视化和Alpha高度值]
 * - Output Type (输出类型): [CMOT Float 4]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [TexCoord]    | Type suggestion: TextureCoordinate (Default: UV0)
 * Pin 1 Name: [TimeInput]   | Type suggestion: Time Node
 * Pin 2 Name: [GridScale]   | Type suggestion: Scalar (Default: 3.0)
 * Pin 3 Name: [Speed]       | Type suggestion: Scalar (Default: 2.0)
 * Pin 4 Name: [Frequency]   | Type suggestion: Scalar (Default: 40.0)
 * =================================================================================
 */

// --- [CODE BODY START] ---

float2 uvScaled = TexCoord * GridScale;
float2 gv = frac(uvScaled) - 0.5;
float2 id = floor(uvScaled);

float randomHash = frac(sin(dot(id, float2(12.9898, 78.233))) * 43758.5453);
float2 posOffset = float2(randomHash - 0.5, frac(randomHash * 2.0) - 0.5) * 0.5;
float dist = length(gv - posOffset);

float t = TimeInput * Speed + randomHash * 6.2831;
float ripple = sin(dist * Frequency - t) * exp(-dist * 8.0);
float gridMask = 1.0 - smoothstep(0.3, 0.5, dist);
float timePulse = 0.5 + 0.5 * sin(t * 0.5); 

float finalHeight = ripple * gridMask * timePulse;

return float4(float3(finalHeight, finalHeight, finalHeight), finalHeight);

// --- [CODE BODY END] ---`
  },
  breathing: {
    name: "Breathing Light",
    code: `/* =================================================================================
 * [Unreal Material Custom Node Template]
 * ---------------------------------------------------------------------------------
 * 1. CONFIGURATION (节点属性设置)
 * - Description (描述): [简单呼吸灯]
 * - Output Type (输出类型): [CMOT Float 4]
 *
 * 2. INPUTS (需要在 Custom 节点上添加的引脚)
 * 注意：引脚名称必须与下方代码中的变量名完全一致
 * ------------------------------------------------------------------------------
 * Pin 0 Name: [TimeInput]   | Type suggestion: Time Node
 * Pin 1 Name: [Speed]       | Type suggestion: Scalar (Default: 2.0)
 * Pin 2 Name: [Intensity]   | Type suggestion: Scalar (Default: 50.0)
 * =================================================================================
 */

// --- [CODE BODY START] ---

float x = TimeInput * Speed;
float blink = (exp(sin(x)) - 0.367879) / 2.3504;

float timeStep = floor(TimeInput * (Speed * 0.16)); 
float3 rndCol;
rndCol.r = frac(sin(dot(float2(timeStep, 12.9), float2(12.9, 78.2))) * 43758.5);
rndCol.g = frac(sin(dot(float2(timeStep, 78.2), float2(26.2, 124.5))) * 143758.5);
rndCol.b = frac(sin(dot(float2(timeStep, 123.5), float2(53.2, 12.0))) * 243758.5);

return float4(rndCol * blink * Intensity, blink);

// --- [CODE BODY END] ---`
  }
};

const UENodeGenerator: React.FC = () => {
  const [hlslCode, setHlslCode] = useState(TEMPLATES.raindrop.code);
  const [inputs, setInputs] = useState<NodeInput[]>([]);
  const [outputType, setOutputType] = useState<OutputType>('CMOT_Float3');
  const [output, setOutput] = useState('');
  const [copied, setCopied] = useState(false);
  const [materialName, setMaterialName] = useState('M_CustomNode');
  const [routing, setRouting] = useState<MaterialOutputSlot[]>([]);

  // Prompt Helper State
  const [showPrompt, setShowPrompt] = useState(false);
  const [promptCopied, setPromptCopied] = useState(false);

  // --- Logic ---

  // Parse HLSL for Template Block
  const parseTemplate = (code: string) => {
    const extracted: NodeInput[] = [];
    
    // 1. Detect Inputs
    // Standard Template Regex: "Pin X Name: [Name] | Type suggestion: Type (Default: Val)"
    const pinRegex = /Pin\s+\d+\s+Name:\s*\[([^\]]+)\]\s*\|\s*Type suggestion:\s*([^\n|(]+)(?:\s*\(Default:\s*([^)]+)\))?/gi;
    let match;
    
    // Only search in comment blocks to avoid false positives in code
    const commentBlockMatch = code.match(/\/\*[\s\S]*?\*\//);
    if (commentBlockMatch) {
      const header = commentBlockMatch[0];
      
      while ((match = pinRegex.exec(header)) !== null) {
        const name = match[1].trim();
        const typeSuggestion = match[2].trim().toLowerCase();
        const defaultValue = match[3] ? match[3].trim() : '';

        let type: ParamType = 'scalar';
        if (typeSuggestion.includes('world') && typeSuggestion.includes('position')) type = 'worldposition';
        else if (typeSuggestion.includes('time')) type = 'time';
        else if (typeSuggestion.includes('texture') || typeSuggestion.includes('coord') || typeSuggestion.includes('uv')) type = 'uv';
        else if (typeSuggestion.includes('vector') || typeSuggestion.includes('color')) type = 'vector';

        extracted.push({
          id: generateGuid(),
          name,
          type,
          defaultValue
        });
      }

      // 2. Detect Output Type
      // Regex: "- Output Type (输出类型): [CMOT Float 4]"
      const outputTypeRegex = /Output Type\s*\(输出类型\):\s*\[(.*?)\]/i;
      const typeMatch = header.match(outputTypeRegex);
      if (typeMatch) {
        let rawType = typeMatch[1].trim(); 
        
        // Fix normalization logic for spaces
        // 1. Handle space between Float and Number (e.g., "Float 4" -> "Float4")
        rawType = rawType.replace(/Float\s+(\d)/i, 'Float$1');
        // 2. Handle space between CMOT and Float (e.g., "CMOT Float" -> "CMOT_Float")
        rawType = rawType.replace(/\s+/g, '_');

        if (['CMOT_Float1', 'CMOT_Float2', 'CMOT_Float3', 'CMOT_Float4'].includes(rawType)) {
             setOutputType(rawType as OutputType);
        }
      }
    }
    
    if (extracted.length > 0) {
      setInputs(extracted);
    }
  };

  // Run parse only once on mount or when template changes explicitly via buttons
  useEffect(() => {
    parseTemplate(hlslCode);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); 

  const generateT3D = () => {
    return buildT3D(hlslCode, inputs, outputType, routing, materialName);
  };

  useEffect(() => {
    setOutput(generateT3D());
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hlslCode, inputs, outputType, routing, materialName]);

  const copyToClipboard = () => {
    if (!output) return;
    navigator.clipboard.writeText(output).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  const copyPrompt = () => {
    navigator.clipboard.writeText(SYSTEM_PROMPT).then(() => {
      setPromptCopied(true);
      setTimeout(() => setPromptCopied(false), 2000);
    });
  };

  const handleManualAdd = () => {
    setInputs([...inputs, { id: generateGuid(), name: 'NewInput', type: 'scalar', defaultValue: '0' }]);
  };

  const handleClearInputs = () => {
    setInputs([]);
  };

  const handleRemoveInput = (id: string) => {
    setInputs(inputs.filter(i => i.id !== id));
  };

  const handleInputChange = (id: string, field: keyof NodeInput, value: string) => {
    setInputs(inputs.map(i => {
      if (i.id !== id) return i;
      if (field === 'useRGBMask') return { ...i, useRGBMask: value === 'true' };
      return { ...i, [field]: value };
    }));
  };

  const handleForceRescan = () => {
    // Clear inputs immediately to signal refresh
    setInputs([]);
    // Slight delay to ensure state clears before parsing again
    setTimeout(() => {
        parseTemplate(hlslCode);
    }, 50);
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 h-full min-h-[600px]">
      
      {/* --- Left Column: Code & Config --- */}
      <div className="flex flex-col gap-4 h-full min-h-0">
        
        {/* AI Prompt Helper Section */}
        <div className="bg-indigo-50 dark:bg-indigo-900/20 border border-indigo-100 dark:border-indigo-500/20 rounded-xl overflow-hidden transition-all">
            <button 
                onClick={() => setShowPrompt(!showPrompt)}
                className="w-full flex items-center justify-between p-3 text-xs font-bold uppercase tracking-wider text-indigo-700 dark:text-indigo-300 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 transition-colors"
            >
                <div className="flex items-center gap-2">
                    <IconSparkles className="w-4 h-4" />
                    <span>AI Shader Generator Prompt</span>
                </div>
                <IconArrowRight className={`w-3 h-3 transition-transform duration-300 ${showPrompt ? '-rotate-90' : 'rotate-90'}`} />
            </button>
            
            {showPrompt && (
                <div className="p-4 border-t border-indigo-100 dark:border-indigo-500/20 bg-white/50 dark:bg-black/20">
                    <p className="text-xs text-gray-500 dark:text-white/60 mb-2">
                        Copy this prompt to ChatGPT or Gemini to generate valid code for this tool.
                    </p>
                    <div className="relative">
                        <textarea 
                            readOnly
                            value={SYSTEM_PROMPT}
                            className="w-full h-32 p-3 text-[10px] font-mono bg-white dark:bg-black/40 border border-gray-200 dark:border-white/10 rounded-lg resize-y focus:outline-none text-gray-600 dark:text-gray-400"
                        />
                        <button 
                            onClick={copyPrompt}
                            className={`absolute top-2 right-2 px-2 py-1 text-[10px] rounded shadow-sm transition-all ${promptCopied ? 'bg-green-500 text-white' : 'bg-white dark:bg-white/10 text-gray-600 dark:text-white border border-gray-200 dark:border-white/10 hover:bg-gray-50'}`}
                        >
                            {promptCopied ? 'Copied' : 'Copy Prompt'}
                        </button>
                    </div>
                </div>
            )}
        </div>

        {/* Toolbar */}
        <div className="flex items-center justify-between">
           <div className="flex gap-2">
              <button 
                 onClick={() => { setHlslCode(TEMPLATES.raindrop.code); setTimeout(() => parseTemplate(TEMPLATES.raindrop.code), 10); }}
                 className="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-100 dark:bg-white/5 hover:bg-gray-200 dark:hover:bg-white/10 text-gray-700 dark:text-white transition-colors"
              >
                 Raindrop
              </button>
              <button 
                 onClick={() => { setHlslCode(TEMPLATES.breathing.code); setTimeout(() => parseTemplate(TEMPLATES.breathing.code), 10); }}
                 className="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-100 dark:bg-white/5 hover:bg-gray-200 dark:hover:bg-white/10 text-gray-700 dark:text-white transition-colors"
              >
                 Breathing
              </button>
           </div>
           <button 
             onClick={handleForceRescan}
             className="text-xs text-indigo-500 hover:text-indigo-400 font-medium flex items-center gap-1"
           >
             <span>Force Rescan</span>
             <span className="w-1.5 h-1.5 rounded-full bg-indigo-500 animate-pulse"></span>
           </button>
        </div>

        {/* Code Editor */}
        <div className="flex-1 relative rounded-xl border border-gray-200 dark:border-white/10 bg-white dark:bg-black/40 overflow-hidden shadow-inner dark:shadow-none focus-within:border-indigo-500/50 transition-colors flex flex-col min-h-[300px]">
            <div className="absolute top-0 right-0 p-2 z-10">
               <IconCode className="w-4 h-4 text-gray-300 dark:text-white/20" />
            </div>
            <textarea 
               value={hlslCode}
               onChange={e => setHlslCode(e.target.value)}
               className="flex-1 w-full p-4 bg-transparent border-none text-gray-800 dark:text-white/90 font-mono text-xs resize-none focus:outline-none leading-relaxed"
               placeholder="// Paste your HLSL shader code here..."
               spellCheck={false}
            />
        </div>

      </div>

      {/* --- Right Column: Parameters & Output --- */}
      <div className="flex flex-col gap-6 h-full min-h-0">
         
         {/* Configuration Section */}
         <div className="flex flex-col gap-3">
             <div className="flex justify-between items-center bg-gray-50 dark:bg-white/5 p-2 rounded-lg">
                 <label className="text-xs uppercase tracking-widest text-gray-500 dark:text-white/40 font-bold pl-2">
                    Node Configuration
                 </label>
             </div>
             <div className="p-3 bg-gray-50 dark:bg-white/5 rounded-lg border border-gray-100 dark:border-white/5">
                <div className="grid grid-cols-12 gap-2 items-center mb-2">
                    <label className="col-span-4 text-xs font-medium text-gray-600 dark:text-white/70">Material Name</label>
                    <div className="col-span-8">
                         <input
                           type="text"
                           value={materialName}
                           onChange={(e) => setMaterialName(e.target.value)}
                           className="w-full bg-white dark:bg-black/20 rounded px-2 py-1.5 text-xs text-gray-700 dark:text-white border border-gray-200 dark:border-white/10 outline-none font-mono"
                           placeholder="M_CustomNode"
                         />
                    </div>
                </div>
                <div className="grid grid-cols-12 gap-2 items-center">
                    <label className="col-span-4 text-xs font-medium text-gray-600 dark:text-white/70">Output Type</label>
                    <div className="col-span-8">
                         <select
                           value={outputType}
                           onChange={(e) => setOutputType(e.target.value as OutputType)}
                           className="w-full bg-white dark:bg-black/20 rounded px-2 py-1.5 text-xs text-gray-700 dark:text-white border border-gray-200 dark:border-white/10 outline-none cursor-pointer"
                         >
                            <option value="CMOT_Float1">Float 1 (Grayscale)</option>
                            <option value="CMOT_Float2">Float 2 (UV/Vec2)</option>
                            <option value="CMOT_Float3">Float 3 (RGB/Pos)</option>
                            <option value="CMOT_Float4">Float 4 (RGBA)</option>
                         </select>
                    </div>
                </div>
             </div>
         </div>

         {/* Parameter List */}
         <div className="flex-1 flex flex-col gap-3 min-h-[200px] overflow-hidden">
             <div className="flex justify-between items-center bg-gray-50 dark:bg-white/5 p-2 rounded-lg">
                 <label className="text-xs uppercase tracking-widest text-gray-500 dark:text-white/40 font-bold pl-2">
                    Node Inputs ({inputs.length})
                 </label>
                 <div className="flex items-center gap-2">
                    {inputs.length > 0 && (
                        <button 
                            onClick={handleClearInputs} 
                            className="text-[10px] text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 px-2 py-1"
                            title="Clear all inputs"
                        >
                            Clear All
                        </button>
                    )}
                    <button onClick={handleManualAdd} className="text-[10px] bg-white dark:bg-white/10 px-2 py-1 rounded border border-gray-200 dark:border-white/10 hover:bg-gray-100 dark:hover:bg-white/20 dark:text-white transition-colors">
                        + Add Input
                    </button>
                 </div>
             </div>
             
             <div className="flex-1 overflow-y-auto space-y-2 pr-2">
                 {inputs.length === 0 ? (
                    <div className="h-full flex flex-col items-center justify-center text-gray-400 dark:text-white/20 text-xs italic border border-dashed border-gray-200 dark:border-white/10 rounded-xl">
                       No inputs detected. <br/>Add manually or Paste Template.
                    </div>
                 ) : (
                    inputs.map((input) => (
                       <div key={input.id} className="grid grid-cols-12 gap-2 p-3 rounded-lg bg-gray-50 dark:bg-white/5 border border-gray-100 dark:border-white/5 items-center group">
                          {/* Name */}
                          <div className="col-span-4">
                             <input 
                               type="text" 
                               value={input.name} 
                               onChange={e => handleInputChange(input.id, 'name', e.target.value)}
                               className="w-full bg-transparent text-sm font-bold text-gray-800 dark:text-white focus:outline-none"
                               placeholder="Name"
                             />
                          </div>
                          {/* Type */}
                          <div className="col-span-3">
                             <select
                               value={input.type}
                               onChange={e => handleInputChange(input.id, 'type', e.target.value as ParamType)}
                               className="w-full bg-black/5 dark:bg-black/20 rounded px-1 py-1 text-xs text-gray-600 dark:text-white/70 border-none outline-none cursor-pointer"
                             >
                                <option value="scalar">Scalar</option>
                                <option value="vector">Vector</option>
                                <option value="time">Time</option>
                                <option value="uv">TexCoord</option>
                                <option value="worldposition">WorldPos</option>
                             </select>
                          </div>
                          {/* Default */}
                          <div className="col-span-4">
                             <div className="flex items-center gap-1">
                                <input
                                  type="text"
                                  value={input.defaultValue}
                                  onChange={e => handleInputChange(input.id, 'defaultValue', e.target.value)}
                                  className="flex-1 min-w-0 bg-transparent text-xs text-gray-500 dark:text-white/50 focus:outline-none text-right font-mono"
                                  placeholder="Default"
                                  disabled={input.type === 'time' || input.type === 'uv' || input.type === 'worldposition'}
                                />
                                {input.type === 'vector' && (
                                  <label className="flex items-center gap-0.5 shrink-0 cursor-pointer" title="Use RGB Mask (float3 input)">
                                    <input
                                      type="checkbox"
                                      checked={!!input.useRGBMask}
                                      onChange={e => handleInputChange(input.id, 'useRGBMask', e.target.checked ? 'true' : '')}
                                      className="w-3 h-3 accent-indigo-500"
                                    />
                                    <span className="text-[9px] text-gray-400 dark:text-white/40">RGB</span>
                                  </label>
                                )}
                             </div>
                          </div>
                          {/* Delete */}
                          <div className="col-span-1 flex justify-end">
                             <button onClick={() => handleRemoveInput(input.id)} className="text-gray-300 hover:text-red-500 dark:text-white/20 dark:hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity">
                                <IconX className="w-3 h-3" />
                             </button>
                          </div>
                       </div>
                    ))
                 )}
             </div>
         </div>

         {/* Action Area */}
         <div className="bg-gray-50 dark:bg-white/5 rounded-2xl p-6 border border-gray-200 dark:border-white/10 flex flex-col gap-4">
             <div className="flex items-start gap-4">
                 <div className="p-3 bg-indigo-100 text-indigo-600 dark:bg-indigo-500/20 dark:text-indigo-400 rounded-xl">
                    <IconUnreal className="w-6 h-6" />
                 </div>
                 <div>
                    <h3 className="text-sm font-bold text-gray-900 dark:text-white">One-Click Generator</h3>
                    <p className="text-xs text-gray-500 dark:text-white/50 mt-1 leading-relaxed">
                       Generates the Custom Node, Input Parameters, and wires them together automatically.
                    </p>
                 </div>
             </div>
             
             <button 
                onClick={copyToClipboard}
                className={`
                   w-full py-4 rounded-xl font-bold uppercase tracking-widest text-xs transition-all duration-300 flex items-center justify-center gap-3 shadow-lg group relative overflow-hidden
                   ${copied 
                       ? 'bg-green-500 text-white shadow-green-500/20 scale-[0.98]' 
                       : 'bg-gray-900 text-white hover:bg-black hover:shadow-xl dark:bg-white dark:text-black dark:hover:bg-gray-200 dark:hover:shadow-white/10 hover:-translate-y-0.5'
                   }
                `}
             >
                <span className="relative z-10 flex items-center gap-2">
                    {copied ? (
                        <>
                           <IconCheck className="w-4 h-4" />
                           <span>Copied to Clipboard</span>
                        </>
                    ) : (
                        <>
                           <IconCopy className="w-4 h-4" />
                           <span>Copy Graph Snippet</span>
                        </>
                    )}
                </span>
             </button>
             <p className="text-center text-[10px] text-gray-400 dark:text-white/30">
                Paste into Unreal Material Editor (Ctrl+V)
             </p>
         </div>

      </div>
    </div>
  );
};

export default UENodeGenerator;