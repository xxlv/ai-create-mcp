package oas31

import (
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxlv/ai-create-mcp/internal/adapters/core"
)

type OAS31Adapter struct {
	oasPath string
}

func New(oasPath string) *OAS31Adapter {
	return &OAS31Adapter{
		oasPath: oasPath,
	}
}
func (a *OAS31Adapter) ToTemplateData() (*core.TemplateData, error) {

	var doc *openapi3.T
	var err error
	if strings.HasPrefix(a.oasPath, "http") || strings.HasPrefix(a.oasPath, "https") {
		uri, parseErr := url.Parse(a.oasPath)
		if parseErr != nil {
			return nil, parseErr
		}
		doc, err = openapi3.NewLoader().LoadFromURI(uri)
	} else {
		doc, err = openapi3.NewLoader().LoadFromFile(a.oasPath)
	}
	if err != nil {
		return nil, err
	}
	return convert(doc)
}

func (a *OAS31Adapter) GetSourceType() string {
	return "oas31"
}

var _ core.Adapter = new(OAS31Adapter)
