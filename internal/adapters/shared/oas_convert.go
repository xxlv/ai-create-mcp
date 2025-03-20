package shared

import (
	"fmt"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxlv/ai-create-mcp/internal/adapters/core"
)

// generateToolName generates a semantic tool name from HTTP method and path.
// For example: "GET" and "/user/{username}" -> "get_user_by_username"
// or "DELETE" and "store_order_{orderId}" -> "delete_store_order_by_orderId"
func generateToolName(method, path string) string {
	method = strings.ToLower(method)
	cleanPath := strings.TrimPrefix(path, "/")
	if cleanPath == "" {
		return method
	}
	cleanPath = strings.ReplaceAll(cleanPath, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "_{", "_by_")
	cleanPath = strings.ReplaceAll(cleanPath, "{", "_by_")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")

	return method + "_" + cleanPath
}

func Convert(doc *openapi3.T) (*core.TemplateData, error) {
	if doc == nil || doc.Info == nil {
		return nil, fmt.Errorf("must provide oas doc")
	}
	missBaseUrl := false
	if doc.Servers == nil {
		missBaseUrl = true
	}
	var endpoints []string
	for _, server := range doc.Servers {
		endpoints = append(endpoints, server.URL)
	}

	data := &core.TemplateData{
		MissBaseURL:   missBaseUrl,
		ServerName:    doc.Info.Title,
		ServerVersion: doc.Info.Version,
		Endpoints:     endpoints, // multiple endpoints
	}
	if len(doc.Servers) > 1 {
		fmt.Printf("WARN: mutlple servers found in oas config file,current just pick the frist!\n")
	}
	for path, pathItem := range doc.Paths.Map() {
		cleanPath := strings.TrimPrefix(path, "/")
		cleanPath = strings.ReplaceAll(cleanPath, "/", "_")

		for method, operation := range pathItem.Operations() {
			opName := generateToolName(method, cleanPath)
			description := operation.Summary
			if description == "" {
				description = operation.Description
			}
			if description == "" {
				description = fmt.Sprintf("%s operation on %s", method, path)
			}

			var arguments []core.Argument
			for _, param := range operation.Parameters {
				arg := core.Argument{
					Name:        safe(param.Value.Name),
					Description: safeDesc(param.Value.Description),
					Required:    param.Value.Required,
				}
				arguments = append(arguments, arg)
			}

			if method == "GET" {
				mimeType := "text/plain"
				if operation.Responses != nil {
					if resp, ok := operation.Responses.Map()["200"]; ok && resp.Value != nil {
						for contentType := range resp.Value.Content {
							mimeType = contentType
							break
						}
					}
				}
				resource := core.Resource{
					Name:        safe(fmt.Sprintf("Resource: %s", cleanPath)),
					Description: safeDesc(description),
					URI:         fmt.Sprintf("ai-create-mcp://internal/%s", cleanPath),
					MimeType:    mimeType,
				}
				data.Resources = append(data.Resources, resource)

				prompt := core.Prompt{
					Name:        safe(opName),
					Description: description,
					Arguments:   arguments,
				}
				data.Prompts = append(data.Prompts, prompt)
			}

			if method == "GET" || method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
				if operation.RequestBody != nil && operation.RequestBody.Value != nil {
					for _, content := range operation.RequestBody.Value.Content {
						if content.Schema != nil && content.Schema.Value != nil {
							for propName, prop := range content.Schema.Value.Properties {
								arg := core.Argument{
									Name:        safe(propName),
									Description: safeDesc(prop.Value.Description),
									Required:    contains(prop.Value.Required, propName),
								}
								arguments = append(arguments, arg)
							}
						}
						break
					}
				}

				tool := core.Tool{
					Name:        safe(opName),
					Description: safeDesc(description),
					Arguments:   arguments,
					Method:      method,
					Path:        path,
				}
				data.Tools = append(data.Tools, tool)
			}
		}
	}
	return data, nil
}

func safe(name string) string {

	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, " ", "")

	return name
}
func safeDesc(name string) string {
	name = strings.ReplaceAll(name, "\"", "")
	return name
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
