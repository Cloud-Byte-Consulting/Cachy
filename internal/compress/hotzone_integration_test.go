package compress

import (
	"strings"
	"testing"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/tokens"
)

func TestOpenAIHotZonesRemainStableWhenNativeCompressionRuns(t *testing.T) {
	t.Parallel()

	system := "stable system prefix with cache-sensitive policy"
	user := "stable user task that should remain byte-for-byte semantically stable"
	assistant := "stable assistant reasoning summary"
	tool := strings.Repeat("2026-06-16T20:00:01Z WARN retrying request\n", 10) +
		"2026-06-16T20:00:03Z ERROR request failed permanently"
	body := []byte(`{
		"model":"gpt-test",
		"messages":[
			{"role":"system","content":` + quoteJSON(system) + `},
			{"role":"user","content":` + quoteJSON(user) + `},
			{"role":"assistant","content":` + quoteJSON(assistant) + `},
			{"role":"tool","tool_call_id":"call_1","content":` + quoteJSON(tool) + `}
		]
	}`)

	blocks, err := DetectLiveZones(Request{Provider: ProviderOpenAI, Body: body})
	if err != nil {
		t.Fatalf("DetectLiveZones() error = %v", err)
	}

	result, err := nativePipeline(ProviderOpenAI).Apply(blocks)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	assertBlockText(t, result.Blocks, "$.messages[0].content", system)
	assertBlockText(t, result.Blocks, "$.messages[1].content", user)
	assertBlockText(t, result.Blocks, "$.messages[2].content", assistant)
	assertSelectedBlockCompressed(t, result, "$.messages[3].content", tool)
}

func TestAnthropicHotZonesRemainStableWhenNativeCompressionRuns(t *testing.T) {
	t.Parallel()

	system := "stable anthropic system prompt"
	user := "stable user content that should not be compressed"
	tool := "{\n  \"status\": \"failed\",\n  \"error\": \"timeout\",\n  \"retryable\": true\n}"
	body := []byte(`{
		"model":"claude-test",
		"system":` + quoteJSON(system) + `,
		"messages":[
			{"role":"user","content":[{"type":"text","text":` + quoteJSON(user) + `}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":` + quoteJSON(tool) + `}]}
		]
	}`)

	blocks, err := DetectLiveZones(Request{Provider: ProviderAnthropic, Body: body})
	if err != nil {
		t.Fatalf("DetectLiveZones() error = %v", err)
	}

	result, err := nativePipeline(ProviderAnthropic).Apply(blocks)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	assertBlockText(t, result.Blocks, "$.system", system)
	assertBlockText(t, result.Blocks, "$.messages[0].content[0].text", user)
	assertSelectedBlockCompressed(t, result, "$.messages[1].content[0].content", tool)
}

func nativePipeline(provider Provider) *Pipeline {
	counters := tokens.NewCounterSet()
	counters.RegisterExact(string(provider), "integration", tokens.CounterFunc(func(text string) (tokens.Count, error) {
		return tokens.Count{Tokens: len(text), Method: "fixture_chars"}, nil
	}))
	return NewPipeline(PipelineOptions{
		Counters: counters,
		ProviderModel: tokens.ProviderModel{
			Provider: string(provider),
			Model:    "integration",
		},
		Compressor: NativeCompressor{},
	})
}

func assertBlockText(t *testing.T, blocks []Block, path, want string) {
	t.Helper()

	for _, block := range blocks {
		if block.Path == path {
			if block.Text != want {
				t.Fatalf("block %s text = %q, want %q", path, block.Text, want)
			}
			if block.Selected {
				t.Fatalf("block %s selected = true, want protected", path)
			}
			return
		}
	}
	t.Fatalf("block %s not found", path)
}

func assertSelectedBlockCompressed(t *testing.T, result PipelineResult, path, original string) {
	t.Helper()

	for i, block := range result.Blocks {
		if block.Path == path {
			if !block.Selected {
				t.Fatalf("block %s selected = false, want live block selected", path)
			}
			if block.Text == original {
				t.Fatalf("block %s was not compressed", path)
			}
			if !result.Decisions[i].Applied {
				t.Fatalf("decision for %s = %#v, want applied", path, result.Decisions[i])
			}
			return
		}
	}
	t.Fatalf("block %s not found", path)
}
