//go:build integration

package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/morphy76/tacito-square/internal/keeper"
	httpAdapter "github.com/morphy76/tacito-square/internal/keeper/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OpenAPI represents the top-level structure of our openapi.json
type OpenAPI struct {
	Paths      map[string]map[string]interface{} `json:"paths"`
	Components struct {
		Schemas map[string]map[string]interface{} `json:"schemas"`
	} `json:"components"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func TestOpenAPIContract_Parity(t *testing.T) {
	// 1. Read openapi.json spec
	specPath := "../../api/openapi/openapi.json"
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "failed to read openapi.json at %s", specPath)

	var spec OpenAPI
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "failed to parse openapi.json")

	// 2. Bootstrap Gin router (nil pool is fine)
	router := keeper.NewServer(nil, nil, nil)
	require.NotNil(t, router)

	// Extract registered Gin routes under /api/v1
	ginRoutes := make(map[string]bool)
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1") {
			key := fmt.Sprintf("%s %s", route.Method, normalizeRoutePath(route.Path))
			ginRoutes[key] = true
		}
	}

	// 3. Extract OpenAPI routes
	openapiRoutes := make(map[string]bool)
	for path, pathObj := range spec.Paths {
		// Prepend /api/v1 prefix
		fullPath := "/api/v1" + path
		fullPath = normalizeRoutePath(fullPath)

		for method := range pathObj {
			upperMethod := strings.ToUpper(method)
			key := fmt.Sprintf("%s %s", upperMethod, fullPath)
			openapiRoutes[key] = true
		}
	}

	// Bidirectional Route Parity Checks (No Less, No More)
	t.Run("OpenAPI routes exist in Gin (No Less)", func(t *testing.T) {
		for routeKey := range openapiRoutes {
			assert.True(t, ginRoutes[routeKey], "Route defined in OpenAPI does not exist in Gin router: %s", routeKey)
		}
	})

	t.Run("Gin routes exist in OpenAPI (No More)", func(t *testing.T) {
		for routeKey := range ginRoutes {
			assert.True(t, openapiRoutes[routeKey], "Route registered in Gin router does not exist in OpenAPI: %s", routeKey)
		}
	})

	// 4. Bidirectional Schema/Model Parity Checks (No Less, No More)
	schemasToVerify := map[string]interface{}{
		"CreateLLMBindingRequest":       httpAdapter.CreateLLMBindingRequest{},
		"UpdateLLMBindingRequest":       httpAdapter.UpdateLLMBindingRequest{},
		"LLMBinding":                    model.LLMBinding{},
		"CreateMCPServerRequest":        httpAdapter.CreateMCPServerRequest{},
		"UpdateMCPServerRequest":        httpAdapter.UpdateMCPServerRequest{},
		"MCPServer":                     model.MCPServer{},
		"CreateSkillRequest":            httpAdapter.CreateSkillRequest{},
		"UpdateSkillRequest":            httpAdapter.UpdateSkillRequest{},
		"Skill":                         model.Skill{},
		"CreateSkillCollectionRequest":  httpAdapter.CreateSkillCollectionRequest{},
		"UpdateSkillCollectionRequest":  httpAdapter.UpdateSkillCollectionRequest{},
		"SkillCollection":               model.SkillCollection{},
		"CreatePromptTemplateRequest":   httpAdapter.CreatePromptTemplateRequest{},
		"UpdatePromptTemplateRequest":   httpAdapter.UpdatePromptTemplateRequest{},
		"PromptTemplate":                model.PromptTemplate{},
		"CreatePromptCollectionRequest": httpAdapter.CreateCollectionRequest{},
		"UpdatePromptCollectionRequest": httpAdapter.UpdateCollectionRequest{},
		"PromptCollection":              model.PromptCollection{},
		"CreateAgentRequest":            httpAdapter.CreateAgentRequest{},
		"UpdateAgentRequest":            httpAdapter.UpdateAgentRequest{},
		"Agent":                         model.Agent{},
		"CreateCommunityRequest":        httpAdapter.CreateCommunityRequest{},
		"UpdateCommunityRequest":        httpAdapter.UpdateCommunityRequest{},
		"Community":                     model.Community{},
		"BrainConfig":                   model.BrainConfig{},
		"ShortTermMemoryConfig":         model.ShortTermMemoryConfig{},
		"LongTermMemoryConfig":          model.LongTermMemoryConfig{},
		"MCPClientConfig":               model.MCPClientConfig{},
		"ErrorResponse":                 ErrorResponse{},
		"AgentStatusDetails":            inbound.AgentStatusDetails{},
		"AgentDeploymentResult":         inbound.AgentDeploymentResult{},
		"CommunityDeploymentDetails":    inbound.CommunityDeploymentDetails{},
		"CommunityStatusDetails":        inbound.CommunityStatusDetails{},
	}

	// Bidirectional registry vs spec check (no less, no more schemas)
	t.Run("OpenAPI schema registry parity", func(t *testing.T) {
		for schemaName := range spec.Components.Schemas {
			_, exists := schemasToVerify[schemaName]
			assert.True(t, exists, "Schema '%s' defined in OpenAPI openapi.json is missing from Go verification registry", schemaName)
		}
		for schemaName := range schemasToVerify {
			_, exists := spec.Components.Schemas[schemaName]
			assert.True(t, exists, "Schema '%s' in Go verification registry is missing from OpenAPI openapi.json specs", schemaName)
		}
	})

	// Detailed property checking for each schema
	for schemaName, goStruct := range schemasToVerify {
		openapiSchema, exists := spec.Components.Schemas[schemaName]
		if !exists {
			continue
		}

		t.Run(fmt.Sprintf("Schema Properties Parity: %s", schemaName), func(t *testing.T) {
			verifySchemaProperties(t, schemaName, openapiSchema, reflect.TypeOf(goStruct))
		})
	}
}

// verifySchemaProperties does bidirectional field-to-property and type parity check via Go reflection
func verifySchemaProperties(t *testing.T, schemaName string, openapiSchema map[string]interface{}, goType reflect.Type) {
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	require.Equal(t, reflect.Struct, goType.Kind(), "Go type for schema %s must be a struct", schemaName)

	props, ok := openapiSchema["properties"].(map[string]interface{})
	if !ok {
		// No properties defined (e.g. empty object), skip property checks but ensure struct has no fields
		assert.Equal(t, 0, goType.NumField(), "OpenAPI schema '%s' has no properties, but Go struct has fields", schemaName)
		return
	}

	// Gather Go struct fields with json tags
	goFields := make(map[string]reflect.StructField)
	for i := 0; i < goType.NumField(); i++ {
		field := goType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		// Strip omitempty or other options
		fieldName := strings.Split(jsonTag, ",")[0]

		// Skip dynamic, internal database segregation field 'tenant_id'
		// ONLY if it is not documented in the OpenAPI properties for this specific schema
		if fieldName == "tenant_id" {
			if _, openApiHasTenantID := props["tenant_id"]; !openApiHasTenantID {
				continue
			}
		}

		goFields[fieldName] = field
	}

	// 1. OpenAPI -> Go Parity (No Less)
	for propName, propVal := range props {
		goField, exists := goFields[propName]
		if assert.True(t, exists, "OpenAPI property '%s' of schema '%s' is missing in Go struct '%s'", propName, schemaName, goType.Name()) {
			// Type parity check
			propSpec, isObj := propVal.(map[string]interface{})
			if isObj {
				verifyFieldType(t, propName, schemaName, propSpec, goField.Type)
			}
		}
	}

	// 2. Go -> OpenAPI Parity (No More)
	for fieldName := range goFields {
		_, exists := props[fieldName]
		assert.True(t, exists, "Go struct field '%s' in '%s' is missing from OpenAPI schema '%s'", fieldName, goType.Name(), schemaName)
	}
}

// verifyFieldType validates standard type compatibility between OpenAPI specs and Go types
func verifyFieldType(t *testing.T, propName string, schemaName string, propSpec map[string]interface{}, fieldType reflect.Type) {
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	openAPIType, _ := propSpec["type"].(string)

	switch fieldType.Kind() {
	case reflect.String:
		// A Go string maps to OpenAPI string
		assert.Equal(t, "string", openAPIType, "Property '%s' in schema '%s': Go string type requires OpenAPI type 'string' (got '%s')", propName, schemaName, openAPIType)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Integers map to integer
		assert.Equal(t, "integer", openAPIType, "Property '%s' in schema '%s': Go integer type requires OpenAPI type 'integer' (got '%s')", propName, schemaName, openAPIType)
	case reflect.Float32, reflect.Float64:
		// Floats map to number
		assert.Equal(t, "number", openAPIType, "Property '%s' in schema '%s': Go float type requires OpenAPI type 'number' (got '%s')", propName, schemaName, openAPIType)
	case reflect.Bool:
		// Bools map to boolean
		assert.Equal(t, "boolean", openAPIType, "Property '%s' in schema '%s': Go bool type requires OpenAPI type 'boolean' (got '%s')", propName, schemaName, openAPIType)
	case reflect.Slice:
		// Slices map to array
		assert.Equal(t, "array", openAPIType, "Property '%s' in schema '%s': Go slice type requires OpenAPI type 'array' (got '%s')", propName, schemaName, openAPIType)
	case reflect.Struct:
		if fieldType.String() != "time.Time" && fieldType.String() != "uuid.UUID" {
			// Nested struct represents object/ref in OpenAPI. It might not have an explicit type but rather $ref.
			if openAPIType != "" {
				assert.Equal(t, "object", openAPIType, "Property '%s' in schema '%s': Go nested struct type requires OpenAPI type 'object' (got '%s')", propName, schemaName, openAPIType)
			}
		} else {
			// time.Time and uuid.UUID are serialized to JSON string format
			assert.Equal(t, "string", openAPIType, "Property '%s' in schema '%s': Go time.Time/UUID requires OpenAPI type 'string' (got '%s')", propName, schemaName, openAPIType)
		}
	case reflect.Map:
		// Map maps to object
		assert.Equal(t, "object", openAPIType, "Property '%s' in schema '%s': Go map type requires OpenAPI type 'object' (got '%s')", propName, schemaName, openAPIType)
	}
}

// normalizeRoutePath transforms all parameter representations to a common :param placeholder.
// e.g. /api/v1/agents/{id} -> /api/v1/agents/:param
// e.g. /api/v1/agents/:agent_id -> /api/v1/agents/:param
func normalizeRoutePath(path string) string {
	res := path
	// 1. Replace OpenAPI style {param_name}
	for {
		start := strings.Index(res, "{")
		if start == -1 {
			break
		}
		end := strings.Index(res, "}")
		if end == -1 {
			break
		}
		res = res[:start] + ":param" + res[end+1:]
	}

	// 2. Replace Gin style :param_name.
	// We split by "/" and replace any segment starting with ":" with ":param"
	segments := strings.Split(res, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = ":param"
		}
	}
	return strings.Join(segments, "/")
}
