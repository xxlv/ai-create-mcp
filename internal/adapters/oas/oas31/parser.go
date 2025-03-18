package oas31

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxlv/ai-create-mcp/internal/adapters/core"
	"github.com/xxlv/ai-create-mcp/internal/adapters/shared"
)

func convert(doc *openapi3.T) (*core.TemplateData, error) {
	return shared.Convert(doc)
}
