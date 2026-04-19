export type ParamType = 'scalar' | 'vector' | 'time' | 'uv' | 'worldposition';
export type OutputType = 'CMOT_Float1' | 'CMOT_Float2' | 'CMOT_Float3' | 'CMOT_Float4';

export interface NodeInput {
  id: string;
  name: string;
  type: ParamType;
  defaultValue: string;
  useRGBMask?: boolean;
}

export type MaterialOutputSlot =
  | 'Base Color'
  | 'Opacity'
  | 'Emissive Color'
  | 'Normal'
  | 'Metallic'
  | 'Specular'
  | 'Roughness'
  | 'Ambient Occlusion'
  | 'Refraction';

interface Edge {
  fromGraph: string;
  fromPin: string;
  toGraph: string;
  toPin: string;
}

interface Pin {
  id: string;
  name: string;
  dir: 'In' | 'Out';
  category: string;
  subCategory: string;
  friendlyName?: string;
  isUObjectWrapper: boolean;
  linkedTo: { graph: string; pin: string }[];
}

interface GraphNode {
  graphName: string;
  exprName: string;
  exprClass: string;
  isRoot?: boolean;
  x: number;
  y: number;
  nodeGuid: string;
  exprGuid: string;
  extraBody: string;
  pins: Pin[];
  canRename?: boolean;
}

export const ROOT_PIN_TABLE: { name: string; sub: string }[] = [
  { name: 'Base Color', sub: '5' },
  { name: 'Metallic', sub: '6' },
  { name: 'Specular', sub: '7' },
  { name: 'Roughness', sub: '8' },
  { name: 'Anisotropy', sub: '9' },
  { name: 'Emissive Color', sub: '0' },
  { name: 'Opacity', sub: '1' },
  { name: 'Opacity Mask', sub: '2' },
  { name: 'Normal', sub: '10' },
  { name: 'Tangent', sub: '11' },
  { name: 'World Position Offset', sub: '12' },
  { name: 'World Displacement', sub: '13' },
  { name: 'Tessellation Multiplier', sub: '14' },
  { name: 'Subsurface Color', sub: '15' },
  { name: 'Custom Data 0', sub: '16' },
  { name: 'Custom Data 1', sub: '17' },
  { name: 'Tree Light Info', sub: '30' },
  { name: 'Ambient Occlusion', sub: '18' },
  { name: 'Refraction', sub: '19' },
  { name: 'Customized UV0', sub: '20' },
  { name: 'Customized UV1', sub: '21' },
  { name: 'Customized UV2', sub: '22' },
  { name: 'Customized UV3', sub: '23' },
  { name: 'Customized UV4', sub: '24' },
  { name: 'Customized UV5', sub: '25' },
  { name: 'Customized UV6', sub: '26' },
  { name: 'Customized UV7', sub: '27' },
  { name: 'Pixel Depth Offset', sub: '28' },
  { name: 'Shading Model', sub: '29' },
  { name: 'Material Attributes', sub: '31' }
];

const SCALAR_SLOTS: Set<string> = new Set([
  'Opacity', 'Opacity Mask', 'Metallic', 'Specular',
  'Roughness', 'Anisotropy', 'Ambient Occlusion',
  'Refraction', 'Tessellation Multiplier', 'Pixel Depth Offset'
]);

export const generateGuid = (): string =>
  'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'.replace(/[x]/g, () =>
    (Math.random() * 16 | 0).toString(16).toUpperCase()
  );

function makePin(overrides: Partial<Pin> & { id: string; name: string; dir: 'In' | 'Out' }): Pin {
  return {
    category: '',
    subCategory: '',
    isUObjectWrapper: false,
    linkedTo: [],
    ...overrides
  };
}

function defaultRouting(outputType: OutputType): MaterialOutputSlot[] {
  switch (outputType) {
    case 'CMOT_Float1': return ['Emissive Color'];
    case 'CMOT_Float2': return ['Emissive Color'];
    case 'CMOT_Float3': return ['Base Color'];
    case 'CMOT_Float4': return ['Base Color', 'Opacity'];
  }
}

function parseVectorDefault(value: string): { r: number; g: number; b: number; a: number } {
  let r = 0, g = 0, b = 0, a = 1;
  if (!value) return { r, g, b, a };
  const parts = value.split(',').map(s => parseFloat(s.trim()));
  if (parts.length === 1 && !isNaN(parts[0])) {
    return { r: parts[0], g: parts[0], b: parts[0], a: 1 };
  }
  if (!isNaN(parts[0])) r = parts[0];
  if (!isNaN(parts[1])) g = parts[1];
  if (!isNaN(parts[2])) b = parts[2];
  if (!isNaN(parts[3])) a = parts[3];
  return { r, g, b, a };
}

// --- Serialization ---

function serializePin(p: Pin): string {
  let s = `   CustomProperties Pin (PinId=${p.id}`;
  s += `,PinName="${p.name}"`;
  if (p.friendlyName) s += `,PinFriendlyName=${p.friendlyName}`;
  if (p.dir === 'Out') s += `,Direction="EGPD_Output"`;
  s += `,PinType.PinCategory="${p.category}"`;
  s += `,PinType.PinSubCategory="${p.subCategory}"`;
  s += `,PinType.PinSubCategoryObject=None`;
  s += `,PinType.PinSubCategoryMemberReference=()`;
  s += `,PinType.PinValueType=()`;
  s += `,PinType.ContainerType=None`;
  s += `,PinType.bIsReference=False`;
  s += `,PinType.bIsConst=False`;
  s += `,PinType.bIsWeakPointer=False`;
  s += `,PinType.bIsUObjectWrapper=${p.isUObjectWrapper ? 'True' : 'False'}`;
  if (p.linkedTo.length > 0) {
    const links = p.linkedTo.map(l => `${l.graph} ${l.pin},`).join('');
    s += `,LinkedTo=(${links})`;
  }
  s += `,PersistentGuid=00000000000000000000000000000000`;
  s += `,bHidden=False,bNotConnectable=False`;
  s += `,bDefaultValueIsReadOnly=False,bDefaultValueIsIgnored=False`;
  s += `,bAdvancedView=False,bOrphanedPin=False,)`;
  return s;
}

function serializeNode(node: GraphNode, matName: string): string {
  const lines: string[] = [];

  if (node.isRoot) {
    lines.push(`Begin Object Class=/Script/UnrealEd.MaterialGraphNode_Root Name="${node.graphName}"`);
    lines.push(`   Material=PreviewMaterial'"/Engine/Transient.${matName}"'`);
    lines.push(`   NodePosX=${node.x}`);
    lines.push(`   NodePosY=${node.y}`);
    lines.push(`   NodeGuid=${node.nodeGuid}`);
  } else {
    lines.push(`Begin Object Class=/Script/UnrealEd.MaterialGraphNode Name="${node.graphName}"`);
    lines.push(`   Begin Object Class=/Script/Engine.${node.exprClass} Name="${node.exprName}"`);
    lines.push(`   End Object`);
    lines.push(`   Begin Object Name="${node.exprName}"`);
    if (node.extraBody) lines.push(node.extraBody);
    lines.push(`      MaterialExpressionEditorX=${node.x}`);
    lines.push(`      MaterialExpressionEditorY=${node.y}`);
    lines.push(`      MaterialExpressionGuid=${node.exprGuid}`);
    lines.push(`      Material=PreviewMaterial'"/Engine/Transient.${matName}"'`);
    lines.push(`   End Object`);
    lines.push(`   MaterialExpression=${node.exprClass}'"${node.exprName}"'`);
    lines.push(`   NodePosX=${node.x}`);
    lines.push(`   NodePosY=${node.y}`);
    if (node.canRename) lines.push(`   bCanRenameNode=True`);
    lines.push(`   NodeGuid=${node.nodeGuid}`);
  }

  for (const p of node.pins) {
    lines.push(serializePin(p));
  }
  lines.push(`End Object`);

  return lines.join('\r\n');
}

// --- Build ---

export function buildT3D(
  hlslCode: string,
  inputs: NodeInput[],
  outputType: OutputType,
  routing: MaterialOutputSlot[] = [],
  materialName: string = ""
): string {
  let nodeIndex = 0;
  const nodes: GraphNode[] = [];
  const edges: Edge[] = [];
  const matName = materialName || "M_CustomNode";

  const effectiveRouting = routing.length > 0 ? routing : defaultRouting(outputType);
  const needsBreakOut = outputType === 'CMOT_Float4'
    && effectiveRouting.some(slot => SCALAR_SLOTS.has(slot));

  // --- 1. Root node ---
  const rootGraphName = 'MaterialGraphNode_Root_0';
  const rootPins: Pin[] = ROOT_PIN_TABLE.map(entry => makePin({
    id: generateGuid(),
    name: entry.name,
    dir: 'In',
    category: 'materialinput',
    subCategory: entry.sub,
  }));
  const rootNode: GraphNode = {
    graphName: rootGraphName,
    exprName: '',
    exprClass: '',
    isRoot: true,
    x: 352, y: 528,
    nodeGuid: generateGuid(),
    exprGuid: generateGuid(),
    extraBody: '',
    pins: rootPins,
  };
  nodes.push(rootNode);

  // --- 2. Custom node ---
  const customGraphName = `MaterialGraphNode_${nodeIndex++}`;
  const customExprGuid = generateGuid();
  const customExprName = `MaterialExpressionCustom_${customExprGuid.substring(0, 8)}`;

  const FRIENDLY = 'NSLOCTEXT("MaterialGraphNode", "Space", " ")';

  const customPins: Pin[] = inputs.map(input => makePin({
    id: generateGuid(),
    name: input.name,
    dir: 'In',
    category: 'required',
    isUObjectWrapper: true,
  }));
  const customOutPin = makePin({
    id: generateGuid(),
    name: 'Output',
    dir: 'Out',
    friendlyName: FRIENDLY,
    isUObjectWrapper: true,
  });
  customPins.push(customOutPin);

  const customInputsStr: string[] = [];
  const customNode: GraphNode = {
    graphName: customGraphName,
    exprName: customExprName,
    exprClass: 'MaterialExpressionCustom',
    x: -432, y: 528,
    nodeGuid: generateGuid(),
    exprGuid: customExprGuid,
    extraBody: '',
    pins: customPins,
  };

  // --- 3. Parameter nodes ---
  inputs.forEach((input, i) => {
    const pGraphName = `MaterialGraphNode_${nodeIndex++}`;
    const pExprGuid = generateGuid();
    const x = -800;
    const y = i * 150 - (inputs.length * 75) + 528;

    let exprClass = '';
    let extraBody = '';
    const pPins: Pin[] = [];
    const pOutPinId = generateGuid();

    if (input.type === 'scalar') {
      exprClass = 'MaterialExpressionScalarParameter';
      const parsedVal = (parseFloat(input.defaultValue) || 0).toFixed(6);
      extraBody = `      DefaultValue=${parsedVal}\r\n      ParameterName="${input.name}"`;
      pPins.push(makePin({ id: pOutPinId, name: 'Output', dir: 'Out', friendlyName: FRIENDLY, isUObjectWrapper: true }));

    } else if (input.type === 'vector') {
      exprClass = 'MaterialExpressionVectorParameter';
      const v = parseVectorDefault(input.defaultValue);
      extraBody = `      DefaultValue=(R=${v.r.toFixed(6)},G=${v.g.toFixed(6)},B=${v.b.toFixed(6)},A=${v.a.toFixed(6)})\r\n      ParameterName="${input.name}"`;
      pPins.push(makePin({ id: pOutPinId, name: 'Output', dir: 'Out', category: 'mask', friendlyName: FRIENDLY, isUObjectWrapper: true }));
      pPins.push(makePin({ id: generateGuid(), name: 'Output2', dir: 'Out', category: 'mask', subCategory: 'red', friendlyName: FRIENDLY, isUObjectWrapper: true }));
      pPins.push(makePin({ id: generateGuid(), name: 'Output3', dir: 'Out', category: 'mask', subCategory: 'green', friendlyName: FRIENDLY }));
      pPins.push(makePin({ id: generateGuid(), name: 'Output4', dir: 'Out', category: 'mask', subCategory: 'blue', friendlyName: FRIENDLY }));
      pPins.push(makePin({ id: generateGuid(), name: 'Output5', dir: 'Out', category: 'mask', subCategory: 'alpha', friendlyName: FRIENDLY }));

    } else if (input.type === 'worldposition') {
      exprClass = 'MaterialExpressionWorldPosition';
      pPins.push(makePin({ id: pOutPinId, name: 'Output', dir: 'Out', friendlyName: FRIENDLY, isUObjectWrapper: true }));

    } else if (input.type === 'time') {
      exprClass = 'MaterialExpressionTime';
      pPins.push(makePin({ id: pOutPinId, name: 'Output', dir: 'Out', friendlyName: FRIENDLY, isUObjectWrapper: true }));

    } else if (input.type === 'uv') {
      exprClass = 'MaterialExpressionTextureCoordinate';
      pPins.push(makePin({ id: pOutPinId, name: 'Output', dir: 'Out', friendlyName: FRIENDLY, isUObjectWrapper: true }));
    }

    const pExprName = `${exprClass}_${pExprGuid.substring(0, 8)}`;
    nodes.push({
      graphName: pGraphName,
      exprName: pExprName,
      exprClass,
      x, y,
      nodeGuid: generateGuid(),
      exprGuid: pExprGuid,
      extraBody,
      pins: pPins,
      canRename: input.type === 'scalar' || input.type === 'vector'
    });

    // Inputs() reference: GraphName.ExprName format
    let maskStr = '';
    if (input.type === 'vector' && input.useRGBMask) {
      maskStr = ',Mask=1,MaskR=1,MaskG=1,MaskB=1';
    }
    customInputsStr.push(
      `      Inputs(${i})=(InputName="${input.name}",Input=(Expression=${exprClass}'"${pGraphName}.${pExprName}"'${maskStr}))`
    );

    // Edge: param output -> custom input
    edges.push({
      fromGraph: pGraphName, fromPin: pOutPinId,
      toGraph: customGraphName, toPin: customPins[i].id
    });
  });

  // Build Custom node extraBody
  const escapedCode = hlslCode
    .replace(/\\/g, '\\\\')
    .replace(/\r\n/g, '\\r\\n')
    .replace(/\n/g, '\\r\\n')
    .replace(/"/g, '\\"');
  customNode.extraBody = [
    `      Code="${escapedCode}"`,
    `      OutputType=${outputType}`,
    ...customInputsStr,
    `      Desc="Generated by BuilderToolKit"`
  ].join('\r\n');
  nodes.push(customNode);

  // --- 4. BreakOut node (if needed) ---
  let breakOutNode: GraphNode | null = null;
  const breakOutPinMap: Record<string, string> = {};

  if (needsBreakOut) {
    const boGraphName = `MaterialGraphNode_${nodeIndex++}`;
    const boExprGuid = generateGuid();
    const boExprName = `MaterialExpressionMaterialFunctionCall_${boExprGuid.substring(0, 8)}`;

    const boInputPin = makePin({
      id: generateGuid(),
      name: 'Float4 (V4)',
      dir: 'In',
      category: 'optional',
      isUObjectWrapper: true,
    });
    const boPinR = makePin({ id: generateGuid(), name: 'R', dir: 'Out', isUObjectWrapper: true });
    const boPinG = makePin({ id: generateGuid(), name: 'G', dir: 'Out', isUObjectWrapper: true });
    const boPinB = makePin({ id: generateGuid(), name: 'B', dir: 'Out', isUObjectWrapper: true });
    const boPinA = makePin({ id: generateGuid(), name: 'A', dir: 'Out', isUObjectWrapper: true });

    breakOutPinMap['R'] = boPinR.id;
    breakOutPinMap['G'] = boPinG.id;
    breakOutPinMap['B'] = boPinB.id;
    breakOutPinMap['A'] = boPinA.id;

    const boExtraBody = [
      `      MaterialFunction=MaterialFunction'"/Engine/Functions/Engine_MaterialFunctions02/Utility/BreakOutFloat4Components.BreakOutFloat4Components"'`,
      `      FunctionInputs(0)=(ExpressionInputId=${generateGuid()},Input=(Expression=MaterialExpressionCustom'"${customGraphName}.${customExprName}"',InputName="Float4"))`,
      `      FunctionOutputs(0)=(ExpressionOutputId=${generateGuid()},Output=(OutputName="R"))`,
      `      FunctionOutputs(1)=(ExpressionOutputId=${generateGuid()},Output=(OutputName="G"))`,
      `      FunctionOutputs(2)=(ExpressionOutputId=${generateGuid()},Output=(OutputName="B"))`,
      `      FunctionOutputs(3)=(ExpressionOutputId=${generateGuid()},Output=(OutputName="A"))`,
    ].join('\r\n');

    breakOutNode = {
      graphName: boGraphName,
      exprName: boExprName,
      exprClass: 'MaterialExpressionMaterialFunctionCall',
      x: -96, y: 608,
      nodeGuid: generateGuid(),
      exprGuid: boExprGuid,
      extraBody: boExtraBody + '\r\n' + [
        `      Outputs(0)=(OutputName="R")`,
        `      Outputs(1)=(OutputName="G")`,
        `      Outputs(2)=(OutputName="B")`,
        `      Outputs(3)=(OutputName="A")`,
      ].join('\r\n'),
      pins: [boInputPin, boPinR, boPinG, boPinB, boPinA],
    };
    nodes.push(breakOutNode);

    // Edge: custom output -> breakout input
    edges.push({
      fromGraph: customGraphName, fromPin: customOutPin.id,
      toGraph: boGraphName, toPin: boInputPin.id,
    });
  }

  // --- 5. Routing edges (Custom/BreakOut -> Root) ---
  for (const slot of effectiveRouting) {
    const rootPin = rootPins.find(p => p.name === slot);
    if (!rootPin) continue;

    if (needsBreakOut && breakOutNode && SCALAR_SLOTS.has(slot)) {
      // Scalar slot -> BreakOut.A
      edges.push({
        fromGraph: breakOutNode.graphName, fromPin: breakOutPinMap['A'],
        toGraph: rootGraphName, toPin: rootPin.id,
      });
    } else {
      // Vector slot or no breakout -> Custom output directly
      edges.push({
        fromGraph: customGraphName, fromPin: customOutPin.id,
        toGraph: rootGraphName, toPin: rootPin.id,
      });
    }
  }

  // --- 6. Apply edges (bidirectional LinkedTo) ---
  for (const edge of edges) {
    const fNode = nodes.find(n => n.graphName === edge.fromGraph);
    const tNode = nodes.find(n => n.graphName === edge.toGraph);
    if (!fNode || !tNode) continue;
    const fPin = fNode.pins.find(p => p.id === edge.fromPin);
    const tPin = tNode.pins.find(p => p.id === edge.toPin);
    if (!fPin || !tPin) continue;
    fPin.linkedTo.push({ graph: edge.toGraph, pin: edge.toPin });
    tPin.linkedTo.push({ graph: edge.fromGraph, pin: edge.fromPin });
  }

  // --- 7. Serialize ---
  return nodes.map(n => serializeNode(n, matName)).join('\r\n');
}
