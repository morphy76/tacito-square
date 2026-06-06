package openapi

import _ "embed"

// Spec holds the committed OpenAPI 3.0.3 contract specification JSON.
//go:embed openapi.json
var Spec []byte
