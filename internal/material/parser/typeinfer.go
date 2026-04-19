package parser

import (
	"strings"

	"github.com/radial/uetx/internal/domain"
)

// InferParamType maps a type suggestion string to a ParamType.
// Order of checks matters — first match wins.
func InferParamType(suggestion string) domain.ParamType {
	s := strings.ToLower(strings.TrimSpace(suggestion))
	switch {
	case strings.Contains(s, "world") && strings.Contains(s, "position"):
		return domain.ParamWorldPosition
	case strings.Contains(s, "time"):
		return domain.ParamTime
	case strings.Contains(s, "texture") || strings.Contains(s, "coord") || strings.Contains(s, "uv"):
		return domain.ParamUV
	case strings.Contains(s, "vector") || strings.Contains(s, "color"):
		return domain.ParamVector
	default:
		return domain.ParamScalar
	}
}
