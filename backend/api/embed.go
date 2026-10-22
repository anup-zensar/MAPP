// Package api holds MAPP's API contract, backend/api/openapi.yaml.
//
// The contract is the source of truth for every endpoint (constitution IV): the Go contract tests check
// every request and response against it, and the web app's TypeScript types are generated from it.
package api

import _ "embed"

// OpenAPI is the contract file, embedded so tests can read it wherever they run.
//
//go:embed openapi.yaml
var OpenAPI []byte
