package main

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToolName(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{
			name:     "Simple DELETE with parameter",
			method:   "DELETE",
			path:     "store_order_{orderId}",
			expected: "delete_store_order_by_orderId",
		},
		{
			name:     "Simple GET with parameter",
			method:   "GET",
			path:     "/pet/{petId}/uploadImage",
			expected: "get_pet_by_petId_uploadImage",
		},
		{
			name:     "Simple GET with parameter",
			method:   "GET",
			path:     "/user/{username}",
			expected: "get_user_by_username",
		},
		{
			name:     "POST with multiple parameters",
			method:   "POST",
			path:     "/user/{username}/posts/{post_id}",
			expected: "post_user_by_username_posts_by_post_id",
		},
		{
			name:     "DELETE with single parameter",
			method:   "DELETE",
			path:     "/user/{username}",
			expected: "delete_user_by_username",
		},
		{
			name:     "GET without parameters",
			method:   "GET",
			path:     "/users",
			expected: "get_users",
		},
		{
			name:     "Simple path without parameters",
			method:   "GET",
			path:     "/status",
			expected: "get_status",
		},
		{
			name:     "Root path",
			method:   "GET",
			path:     "/",
			expected: "get",
		},
		{
			name:     "Mixed case method",
			method:   "PoSt",
			path:     "/user/{username}",
			expected: "post_user_by_username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateToolName(tt.method, tt.path)
			if result != tt.expected {
				t.Errorf("generateToolName(%q, %q) = %q, want %q", tt.method, tt.path, result, tt.expected)
			}
		})
	}
}

func TestConvertOAStoTemplateData(t *testing.T) {
	tests := []struct {
		name    string
		doc     *openapi3.T
		want    TemplateData
		wantErr bool
	}{
		{
			name: "document with POST operation",
			doc: &openapi3.T{
				Info: &openapi3.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Servers: []*openapi3.Server{
					{
						URL: "https://test.com",
					},
				},
				Paths: func() *openapi3.Paths {
					paths := openapi3.NewPaths()
					paths.Set("/users", &openapi3.PathItem{
						Post: &openapi3.Operation{
							Summary: "Create user",
							RequestBody: &openapi3.RequestBodyRef{
								Value: &openapi3.RequestBody{
									Content: openapi3.Content{
										"application/json": &openapi3.MediaType{
											Schema: &openapi3.SchemaRef{
												Value: &openapi3.Schema{
													Properties: openapi3.Schemas{
														"name": &openapi3.SchemaRef{
															Value: &openapi3.Schema{
																Description: "User name",
															},
														},
													},
													Required: []string{"name"},
												},
											},
										},
									},
								},
							},
						},
					})
					return paths
				}(),
			},
			want: TemplateData{
				ServerName:    "Test API",
				ServerVersion: "1.0.0",
				Endpoint:      "https://test.com",
				Tools: []Tool{
					{
						Name:        "post_users",
						Description: "Create user",
						Arguments: []Argument{
							{
								Name:        "name",
								Description: "User name",
								Required:    false,
							},
						},
						Method: "POST",
						Path:   "/users",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil document",
			doc:     nil,
			want:    TemplateData{},
			wantErr: true,
		},
		{
			name: "basic document with info",
			doc: &openapi3.T{
				Info: &openapi3.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: openapi3.NewPaths(),
				Servers: []*openapi3.Server{
					{
						URL: "https://test.com",
					},
				},
			},

			want: TemplateData{
				ServerName:    "Test API",
				ServerVersion: "1.0.0",
				Endpoint:      "https://test.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertOAStoTemplateData(tt.doc)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.ServerName, got.ServerName)
			assert.Equal(t, tt.want.ServerVersion, got.ServerVersion)
			assert.ElementsMatch(t, tt.want.Resources, got.Resources)
			assert.ElementsMatch(t, tt.want.Prompts, got.Prompts)
			assert.ElementsMatch(t, tt.want.Tools, got.Tools)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item exists",
			slice: []string{"a", "b", "c"},
			item:  "b",
			want:  true,
		},
		{
			name:  "item doesn't exist",
			slice: []string{"a", "b", "c"},
			item:  "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "a",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}
