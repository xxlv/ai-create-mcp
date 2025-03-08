package main

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func ConvertOAStoTemplateData(doc *openapi3.T) (TemplateData, error) {
	if doc == nil {
		return TemplateData{}, fmt.Errorf("must provide oas doc")
	}
	data := TemplateData{
		ServerName:    doc.Info.Title,
		ServerVersion: doc.Info.Version,
	}

	// Iterate over all paths and operations in the OAS document
	for path, pathItem := range doc.Paths.Map() {
		// Clean path for naming (replace slashes with underscores)
		cleanPath := strings.TrimPrefix(path, "/")
		cleanPath = strings.ReplaceAll(cleanPath, "/", "_")

		// Handle each operation (GET, POST, etc.)
		for method, operation := range pathItem.Operations() {
			opName := strings.ToLower(method) + "_" + cleanPath
			description := operation.Summary
			if description == "" {
				description = operation.Description
			}
			if description == "" {
				description = fmt.Sprintf("%s operation on %s", method, path)
			}

			// Collect arguments from parameters
			var arguments []Argument
			for _, param := range operation.Parameters {
				arg := Argument{
					Name:        param.Value.Name,
					Description: param.Value.Description,
					Required:    param.Value.Required,
				}
				arguments = append(arguments, arg)
			}

			// Handle GET operations (Resources and Prompts)
			if method == "GET" {
				// Add as a Resource
				mimeType := "text/plain" // Default
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

				// Add as a Prompt
				prompt := Prompt{
					Name:        opName,
					Description: description,
					Arguments:   arguments,
				}
				data.Prompts = append(data.Prompts, prompt)
			}

			// Handle POST/PUT/PATCH/DELETE operations (Tools)
			if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
				// Add request body as arguments (if present)
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
						break // Use first content type
					}
				}

				tool := Tool{
					Name:        opName,
					Description: description,
					Arguments:   arguments,
				}
				data.Tools = append(data.Tools, tool)
			}
		}
	}

	return data, nil
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
