# Composable Go API

Cachy's reusable Go surface lives under `pkg/` of the module
`truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy`. Code under `internal/` is still
reserved for the CLI/proxy implementation and can change more freely.

## Public Packages

```text
pkg/proxy          embeddable http.Handler for provider-compatible proxying
pkg/compress       live-zone detection, native compressors, compression pipeline
pkg/tokens         token counting and savings validation
pkg/ccr            content-addressed retrieval markers and local store
pkg/platform       Windows/macOS/Linux config/state/cache path resolution
pkg/observability  privacy-safe telemetry and logging helpers
```

## Embedding The Proxy

```go
package main

import (
	"log"
	"net/http"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/pkg/proxy"
)

func main() {
	handler, err := proxy.NewHandler(proxy.Config{
		TargetBaseURL: "http://127.0.0.1:11434",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe("127.0.0.1:8787", handler))
}
```

## Using Compression Directly

```go
blocks, err := compress.DetectLiveZones(compress.Request{
	Provider: compress.ProviderOpenAI,
	Body:     requestBody,
})
if err != nil {
	return err
}

pipeline := compress.NewPipeline(compress.PipelineOptions{
	Compressor:    compress.NativeCompressor{},
	ProviderModel: tokens.ProviderModel{Provider: "openai", Model: "gpt-4.1"},
})

result, err := pipeline.Apply(blocks)
```

## Using From Another Module

Cachy's module path is its Gitea URL, so consumers `go get` it directly from the
Cloud-Byte-Consulting Gitea — no `replace` directive:

```go
require truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy v0.1.0
```

```sh
go env -w GOPRIVATE=truenas-scale-1.tail5a208d.ts.net   # fetch directly, skip public proxy/sumdb
go get truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy@v0.1.0
```

> The module path embeds the current Tailscale hostname; a stable custom domain
> (CNAME → the Gitea) can be swapped in later via a repo-wide find/replace.

## Compatibility Promise

- `pkg/` packages are the intended composable API surface.
- `internal/` packages are implementation details.
- Additions to `pkg/` should be backward-compatible where practical.
- Breaking API changes should be recorded in an ADR or release note.
