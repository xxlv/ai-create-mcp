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

func ConvertOAStoTemplateData(doc *openapi3.T) (TemplateData, error) {
	empty := TemplateData{}
	if doc == nil {
		return empty, fmt.Errorf("must provide oas doc")
	}
	if doc.Servers == nil || len(doc.Servers) <= 0 {
		return empty, fmt.Errorf("oas must contains server,please check your config file")
	}
	data := TemplateData{
		ServerName:    doc.Info.Title,
		ServerVersion: doc.Info.Version,
		Endpoint:      doc.Servers[0].URL,
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

			var arguments []Argument
			for _, param := range operation.Parameters {
				arg := Argument{
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
				resource := Resource{
					Name:        safe(fmt.Sprintf("Resource: %s", cleanPath)),
					Description: safeDesc(description),
					URI:         fmt.Sprintf("ai-create-mcp://internal/%s", cleanPath),
					MimeType:    mimeType,
				}
				data.Resources = append(data.Resources, resource)

				prompt := Prompt{
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
								arg := Argument{
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

				tool := Tool{
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
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
