package serializer

import (
	"fmt"
	"strings"

	"github.com/radial/uetx/internal/domain"
)

const crlf = "\r\n"

// SerializeGraph converts graph nodes into T3D text with CRLF line endings.
func SerializeGraph(nodes []*domain.GraphNode, materialName string) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = serializeNode(n, materialName)
	}
	return strings.Join(parts, crlf)
}

func serializeNode(node *domain.GraphNode, matName string) string {
	var b strings.Builder

	if node.IsRoot {
		fmt.Fprintf(&b, "Begin Object Class=/Script/UnrealEd.MaterialGraphNode_Root Name=\"%s\"", node.GraphName)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   Material=PreviewMaterial'\"/Engine/Transient.%s\"'", matName)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   NodePosX=%d", node.X)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   NodePosY=%d", node.Y)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   NodeGuid=%s", node.NodeGUID)
	} else {
		fmt.Fprintf(&b, "Begin Object Class=/Script/UnrealEd.MaterialGraphNode Name=\"%s\"", node.GraphName)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   Begin Object Class=/Script/Engine.%s Name=\"%s\"", node.ExprClass, node.ExprName)
		b.WriteString(crlf)
		b.WriteString("   End Object")
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   Begin Object Name=\"%s\"", node.ExprName)
		b.WriteString(crlf)
		if node.ExtraBody != "" {
			b.WriteString(node.ExtraBody)
			b.WriteString(crlf)
		}
		fmt.Fprintf(&b, "      MaterialExpressionEditorX=%d", node.X)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "      MaterialExpressionEditorY=%d", node.Y)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "      MaterialExpressionGuid=%s", node.ExprGUID)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "      Material=PreviewMaterial'\"/Engine/Transient.%s\"'", matName)
		b.WriteString(crlf)
		b.WriteString("   End Object")
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   MaterialExpression=%s'\"%s\"'", node.ExprClass, node.ExprName)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   NodePosX=%d", node.X)
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   NodePosY=%d", node.Y)
		if node.CanRename {
			b.WriteString(crlf)
			b.WriteString("   bCanRenameNode=True")
		}
		b.WriteString(crlf)
		fmt.Fprintf(&b, "   NodeGuid=%s", node.NodeGUID)
	}

	for _, p := range node.Pins {
		b.WriteString(crlf)
		b.WriteString(serializePin(p))
	}

	b.WriteString(crlf)
	b.WriteString("End Object")

	return b.String()
}

func serializePin(p *domain.Pin) string {
	var b strings.Builder
	fmt.Fprintf(&b, "   CustomProperties Pin (PinId=%s", p.ID)
	fmt.Fprintf(&b, ",PinName=\"%s\"", p.Name)
	if p.FriendlyName != "" {
		fmt.Fprintf(&b, ",PinFriendlyName=%s", p.FriendlyName)
	}
	if p.Dir == domain.PinDirOut {
		b.WriteString(",Direction=\"EGPD_Output\"")
	}
	fmt.Fprintf(&b, ",PinType.PinCategory=\"%s\"", p.Category)
	fmt.Fprintf(&b, ",PinType.PinSubCategory=\"%s\"", p.SubCategory)
	b.WriteString(",PinType.PinSubCategoryObject=None")
	b.WriteString(",PinType.PinSubCategoryMemberReference=()")
	b.WriteString(",PinType.PinValueType=()")
	b.WriteString(",PinType.ContainerType=None")
	b.WriteString(",PinType.bIsReference=False")
	b.WriteString(",PinType.bIsConst=False")
	b.WriteString(",PinType.bIsWeakPointer=False")
	fmt.Fprintf(&b, ",PinType.bIsUObjectWrapper=%s", boolStr(p.IsUObjectWrapper))
	if len(p.LinkedTo) > 0 {
		b.WriteString(",LinkedTo=(")
		for _, l := range p.LinkedTo {
			fmt.Fprintf(&b, "%s %s,", l.GraphName, l.PinID)
		}
		b.WriteString(")")
	}
	b.WriteString(",PersistentGuid=00000000000000000000000000000000")
	b.WriteString(",bHidden=False,bNotConnectable=False")
	b.WriteString(",bDefaultValueIsReadOnly=False,bDefaultValueIsIgnored=False")
	b.WriteString(",bAdvancedView=False,bOrphanedPin=False,)")
	return b.String()
}

func boolStr(v bool) string {
	if v {
		return "True"
	}
	return "False"
}
