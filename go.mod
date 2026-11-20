module github.com/joshuasprow/nitrovin

go 1.27

require (
	github.com/andybalholm/brotli v1.3.0
	// chromedp drives headless Chrome over the DevTools protocol for
	// TestForm's browser runs (cmd/nitrovin/browser_test.go): tests only, no
	// binary links it. go-json-experiment below is its own.
	github.com/chromedp/cdproto v0.159.0
	github.com/chromedp/chromedp v0.23.0
	// regexp2 (MIT) reads the published schemas' patterns as ECMA-262, the
	// dialect JSON Schema names, where Go's regexp is RE2
	// (pkg/vocab/schema_test.go): tests only, no binary links it.
	// jsonschema's own tests already required it.
	github.com/dlclark/regexp2 v1.11.0
	// pgx (MIT) reads a stock connector's relation at request time
	// (ADR 0020). Only pkg/stock/pg imports it, and only cmd/nitrovin
	// imports that: the C library and the WebAssembly build carry no
	// driver. pgpassfile, pgservicefile and puddle below are its own, and
	// it raised x/sync and x/text.
	github.com/jackc/pgx/v5 v5.13.0
	github.com/joshuasprow/nitrovin-data v1.202611.1
	// The official MCP SDK serves `nitrovin mcp` (ADR 0005): it carries every
	// protocol revision (2024-11-05 through 2026-07-28, initialize and
	// server/discover) so nitrovin never ports MCP itself. Only cmd/nitrovin
	// imports it; the indirect requirements below from jsonschema-go to x/time
	// are its own.
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	github.com/zeebo/blake3 v0.2.4
	go.yaml.in/yaml/v3 v3.0.6
	// fallback (BSD-3-Clause) carries Mozilla's CA roots for TLS where the
	// system has none: the image is the binary on scratch (#358). Only
	// cmd/nitrovin imports it.
	golang.org/x/crypto/x509roots/fallback v0.0.0-20261115193255-739070949446
)

require (
	github.com/go-json-experiment/json v0.0.0-20260213210345-44df1a37e875 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/klauspost/cpuid/v2 v2.0.12 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)
