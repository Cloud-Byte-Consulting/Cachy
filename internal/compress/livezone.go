package compress

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

type Stability string

const (
	StabilityStable Stability = "stable"
	StabilityLive   Stability = "live"
)

type Source string

const (
	SourceSystem           Source = "system"
	SourceUserMessage      Source = "user_message"
	SourceAssistantMessage Source = "assistant_message"
	SourceToolResult       Source = "tool_result"
)

type ContentType string

const (
	ContentText ContentType = "text"
	ContentLog  ContentType = "log"
	ContentCode ContentType = "code"
	ContentDiff ContentType = "diff"
	ContentJSON ContentType = "json"
)

type Request struct {
	Provider Provider
	Body     []byte
}

type Block struct {
	ID          string
	Provider    Provider
	Path        string
	Role        string
	Source      Source
	Stability   Stability
	ContentType ContentType
	Text        string
	Selected    bool
}

func DetectLiveZones(request Request) ([]Block, error) {
	switch request.Provider {
	case ProviderOpenAI:
		return detectOpenAI(request.Body)
	case ProviderAnthropic:
		return detectAnthropic(request.Body)
	default:
		return nil, fmt.Errorf("unsupported provider %q", request.Provider)
	}
}

type openAIRequest struct {
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func detectOpenAI(body []byte) ([]Block, error) {
	var request openAIRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, err
	}

	blocks := make([]Block, 0, len(request.Messages))
	for i, message := range request.Messages {
		texts := openAIContentText(message.Content)
		for j, text := range texts {
			if text == "" {
				continue
			}
			path := fmt.Sprintf("$.messages[%d].content", i)
			if len(texts) > 1 {
				path = fmt.Sprintf("$.messages[%d].content[%d].text", i, j)
			}
			blocks = append(blocks, newBlock(ProviderOpenAI, path, message.Role, text))
		}
	}
	assignBlockIDs(blocks)
	return blocks, nil
}

func openAIContentText(raw json.RawMessage) []string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []string{text}
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil
	}
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return texts
}

type anthropicRequest struct {
	System   json.RawMessage    `json:"system"`
	Messages []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicPart struct {
	Type    string          `json:"type"`
	Text    string          `json:"text"`
	Content json.RawMessage `json:"content"`
}

func detectAnthropic(body []byte) ([]Block, error) {
	var request anthropicRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, err
	}

	blocks := []Block{}
	for _, text := range anthropicSystemTexts(request.System) {
		if text != "" {
			blocks = append(blocks, Block{
				ID:          "block_1",
				Provider:    ProviderAnthropic,
				Path:        "$.system",
				Role:        "system",
				Source:      SourceSystem,
				Stability:   StabilityStable,
				ContentType: ContentText,
				Text:        text,
				Selected:    false,
			})
		}
	}

	for i, message := range request.Messages {
		parts, err := anthropicContentParts(message.Content)
		if err != nil {
			return nil, err
		}
		for j, part := range parts {
			text := anthropicPartText(part)
			if text == "" {
				continue
			}
			path := fmt.Sprintf("$.messages[%d].content[%d].text", i, j)
			if part.Type == "tool_result" {
				path = fmt.Sprintf("$.messages[%d].content[%d].content", i, j)
			}
			block := newBlock(ProviderAnthropic, path, message.Role, text)
			if part.Type == "tool_result" {
				block.Source = SourceToolResult
				block.Stability = StabilityLive
				block.Selected = true
				block.ContentType = ClassifyContent(text)
			}
			blocks = append(blocks, block)
		}
	}

	for i := range blocks {
		assignBlockID(&blocks[i], i)
	}
	return blocks, nil
}

func assignBlockIDs(blocks []Block) {
	for i := range blocks {
		assignBlockID(&blocks[i], i)
	}
}

func assignBlockID(block *Block, index int) {
	block.ID = fmt.Sprintf("block_%d", index+1)
}

func anthropicSystemTexts(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []string{text}
	}
	var parts []anthropicPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil
	}
	texts := []string{}
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return texts
}

func anthropicContentParts(raw json.RawMessage) ([]anthropicPart, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []anthropicPart{{Type: "text", Text: text}}, nil
	}
	var parts []anthropicPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil, err
	}
	return parts, nil
}

func anthropicPartText(part anthropicPart) string {
	if part.Text != "" {
		return part.Text
	}
	if len(part.Content) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(part.Content, &text); err == nil {
		return text
	}
	var items []anthropicPart
	if err := json.Unmarshal(part.Content, &items); err == nil {
		texts := []string{}
		for _, item := range items {
			if item.Text != "" {
				texts = append(texts, item.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	return string(part.Content)
}

func newBlock(provider Provider, path, role, text string) Block {
	source, stability, selected := classifySource(role)
	return Block{
		Provider:    provider,
		Path:        path,
		Role:        role,
		Source:      source,
		Stability:   stability,
		ContentType: ClassifyContent(text),
		Text:        text,
		Selected:    selected,
	}
}

func classifySource(role string) (Source, Stability, bool) {
	switch role {
	case "system", "developer":
		return SourceSystem, StabilityStable, false
	case "assistant":
		return SourceAssistantMessage, StabilityStable, false
	case "tool":
		return SourceToolResult, StabilityLive, true
	default:
		return SourceUserMessage, StabilityStable, false
	}
}

func ClassifyContent(text string) ContentType {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ContentText
	}
	if json.Valid([]byte(trimmed)) && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
		return ContentJSON
	}
	if looksLikeDiff(trimmed) {
		return ContentDiff
	}
	if looksLikeCode(trimmed) {
		return ContentCode
	}
	if looksLikeLog(trimmed) {
		return ContentLog
	}
	return ContentText
}

func looksLikeDiff(text string) bool {
	return strings.Contains(text, "\n+++") ||
		strings.Contains(text, "\n---") ||
		strings.Contains(text, "\n@@ ") ||
		strings.HasPrefix(text, "diff --git")
}

func looksLikeCode(text string) bool {
	return strings.HasPrefix(text, "```") ||
		strings.Contains(text, "\nfunc ") ||
		strings.Contains(text, "\npackage ") ||
		strings.Contains(text, "\nclass ") ||
		strings.Contains(text, "\ndef ")
}

func looksLikeLog(text string) bool {
	upper := strings.ToUpper(text)
	return strings.Contains(upper, "ERROR") ||
		strings.Contains(upper, "WARN") ||
		strings.Contains(text, "stack trace") ||
		strings.Contains(text, "Traceback")
}

var ErrNoLiveBlocks = errors.New("no live-zone blocks detected")
