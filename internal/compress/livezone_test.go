package compress

import (
	"encoding/json"
	"testing"
)

func TestDetectLiveZonesSelectsOnlyOpenAIToolBlocks(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gpt-4.1",
		"messages":[
			{"role":"system","content":"stable policy prefix"},
			{"role":"user","content":"stable user task"},
			{"role":"assistant","content":"stable assistant plan"},
			{"role":"tool","tool_call_id":"call_1","content":"ERROR failed\n    at main.go:42\n    at main.go:44"}
		]
	}`)

	blocks, err := DetectLiveZones(Request{Provider: ProviderOpenAI, Body: body})
	if err != nil {
		t.Fatalf("DetectLiveZones() error = %v", err)
	}

	if len(blocks) != 4 {
		t.Fatalf("blocks = %d, want 4", len(blocks))
	}
	assertBlock(t, blocks[0], StabilityStable, SourceSystem, ContentText, false)
	assertBlock(t, blocks[1], StabilityStable, SourceUserMessage, ContentText, false)
	assertBlock(t, blocks[2], StabilityStable, SourceAssistantMessage, ContentText, false)
	assertBlock(t, blocks[3], StabilityLive, SourceToolResult, ContentLog, true)
	if blocks[3].ID != "block_4" {
		t.Fatalf("tool block id = %q, want block_4", blocks[3].ID)
	}
	if blocks[3].Path != "$.messages[3].content" {
		t.Fatalf("tool block path = %q, want $.messages[3].content", blocks[3].Path)
	}
}

func TestDetectLiveZonesSelectsAnthropicToolResults(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"claude-3-5-sonnet-latest",
		"system":"stable system prompt",
		"messages":[
			{"role":"user","content":[{"type":"text","text":"stable user task"}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"diff --git a/a.go b/a.go\n+added\n-removed"}]}
		]
	}`)

	blocks, err := DetectLiveZones(Request{Provider: ProviderAnthropic, Body: body})
	if err != nil {
		t.Fatalf("DetectLiveZones() error = %v", err)
	}

	if len(blocks) != 3 {
		t.Fatalf("blocks = %d, want 3", len(blocks))
	}
	assertBlock(t, blocks[0], StabilityStable, SourceSystem, ContentText, false)
	assertBlock(t, blocks[1], StabilityStable, SourceUserMessage, ContentText, false)
	assertBlock(t, blocks[2], StabilityLive, SourceToolResult, ContentDiff, true)
	if blocks[2].Path != "$.messages[1].content[0].content" {
		t.Fatalf("tool result path = %q, want $.messages[1].content[0].content", blocks[2].Path)
	}
}

func TestDetectLiveZonesClassifiesSelectedToolContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want ContentType
	}{
		{name: "json", text: `{"items":[1,2,3],"ok":true}`, want: ContentJSON},
		{name: "diff", text: "--- a/file.go\n+++ b/file.go\n@@ -1 +1 @@\n-old\n+new", want: ContentDiff},
		{name: "code", text: "```go\nfunc main() {}\n```", want: ContentCode},
		{name: "log", text: "2026-06-16T01:00:00Z ERROR request failed\nstack trace line", want: ContentLog},
		{name: "text", text: "ordinary plain text result", want: ContentText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := []byte(`{"messages":[{"role":"tool","content":` + quoteJSON(tt.text) + `}]}`)
			blocks, err := DetectLiveZones(Request{Provider: ProviderOpenAI, Body: body})
			if err != nil {
				t.Fatalf("DetectLiveZones() error = %v", err)
			}
			if len(blocks) != 1 {
				t.Fatalf("blocks = %d, want 1", len(blocks))
			}
			assertBlock(t, blocks[0], StabilityLive, SourceToolResult, tt.want, true)
		})
	}
}

func TestDetectLiveZonesRejectsUnknownProviderAndInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := DetectLiveZones(Request{Provider: "unknown", Body: []byte(`{}`)}); err == nil {
		t.Fatal("DetectLiveZones() unknown provider error = nil, want error")
	}
	if _, err := DetectLiveZones(Request{Provider: ProviderOpenAI, Body: []byte(`not-json`)}); err == nil {
		t.Fatal("DetectLiveZones() invalid JSON error = nil, want error")
	}
}

func assertBlock(t *testing.T, block Block, stability Stability, source Source, content ContentType, selected bool) {
	t.Helper()

	if block.Stability != stability || block.Source != source || block.ContentType != content || block.Selected != selected {
		t.Fatalf("block = %#v, want stability=%s source=%s content=%s selected=%v", block, stability, source, content, selected)
	}
}

func quoteJSON(text string) string {
	data, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return string(data)
}
