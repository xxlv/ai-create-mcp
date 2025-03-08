package main

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// generateToolName generates a semantic tool name from HTTP method and path.
// For example: "GET" and "/user/{username}" -> "get_user_by_username"
// or "DELETE" and "store_order_{orderId}" -> "delete_store_order_by_orderId"
func generateToolName(method, path string) string {
	// Convert method to lowercase
	method = strings.ToLower(method)

	// Clean the path by removing leading slash
	cleanPath := strings.TrimPrefix(path, "/")

	// If path is empty (root path), return method only
	if cleanPath == "" {
		return method
	}

	// Replace "/" with "_", "{" with "_by", and "}" with ""
	cleanPath = strings.ReplaceAll(cleanPath, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "_{", "_by_")
	cleanPath = strings.ReplaceAll(cleanPath, "{", "_by_")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")

	// Combine method and cleaned path
	return method + "_" + cleanPath
}

func ConvertOAStoTemplateData(doc *openapi3.T) (TemplateData, error) {
	if doc == nil {
		return TemplateData{}, fmt.Errorf("must provide oas doc")
	}
	data := TemplateData{
		ServerName:    doc.Info.Title,
		ServerVersion: doc.Info.Version,
		Endpoint:      doc.Servers[0].URL,
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

			var arguments []Argument
			for _, param := range operation.Parameters {
				arg := Argument{
					Name:        param.Value.Name,
					Description: param.Value.Description,
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
				resource := Resource{
					Name:        fmt.Sprintf("Resource: %s", cleanPath),
					Description: description,
					URI:         fmt.Sprintf("ai-create-mcp://internal/%s", cleanPath),
					MimeType:    mimeType,
				}
				data.Resources = append(data.Resources, resource)

				prompt := Prompt{
					Name:        opName,
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
								arg := Argument{
									Name:        propName,
									Description: prop.Value.Description,
									Required:    contains(prop.Value.Required, propName),
								}
								arguments = append(arguments, arg)
							}
						}
						break
					}
				}

				tool := Tool{
					Name:        opName,
					Description: description,
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
