package compress

import (
	"strings"
	"testing"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/tokens"
)

func TestNativeTextLogCompressorReducesRepeatedLogNoise(t *testing.T) {
	t.Parallel()

	logText := strings.Join([]string{
		"2026-06-16T20:00:00Z ERROR request failed: database timeout",
		"    at db/query.go:42",
		"    at db/query.go:42",
		"    at db/query.go:42",
		"2026-06-16T20:00:01Z WARN retrying request",
		"2026-06-16T20:00:01Z WARN retrying request",
		"2026-06-16T20:00:01Z WARN retrying request",
	}, "\n")

	proposal, err := NativeTextLogCompressor{}.Compress(Block{
		ID:          "block_1",
		ContentType: ContentLog,
		Text:        logText,
	})
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if len(proposal.Text) >= len(logText) {
		t.Fatalf("proposal length = %d, want shorter than %d", len(proposal.Text), len(logText))
	}
	for _, want := range []string{"ERROR request failed", "database timeout", "db/query.go:42", "WARN retrying request"} {
		if !strings.Contains(proposal.Text, want) {
			t.Fatalf("proposal missing diagnostic signal %q:\n%s", want, proposal.Text)
		}
	}
	if !strings.Contains(proposal.Text, "repeated") {
		t.Fatalf("proposal does not describe omitted repetition:\n%s", proposal.Text)
	}
}

func TestNativeTextLogCompressorReducesRepeatedTextBlocks(t *testing.T) {
	t.Parallel()

	paragraph := "The build emitted the same advisory for every package and the underlying fix is unchanged."
	text := strings.Join([]string{
		paragraph,
		paragraph,
		paragraph,
		"Final diagnostic: update the lockfile once.",
	}, "\n\n")

	proposal, err := NativeTextLogCompressor{}.Compress(Block{
		ID:          "block_1",
		ContentType: ContentText,
		Text:        text,
	})
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if len(proposal.Text) >= len(text) {
		t.Fatalf("proposal length = %d, want shorter than %d", len(proposal.Text), len(text))
	}
	if strings.Count(proposal.Text, paragraph) != 1 {
		t.Fatalf("proposal = %q, want repeated paragraph kept once", proposal.Text)
	}
	if !strings.Contains(proposal.Text, "Final diagnostic") {
		t.Fatalf("proposal missing final diagnostic: %q", proposal.Text)
	}
}

func TestNativeTextLogCompressorFeedsValidationPipeline(t *testing.T) {
	t.Parallel()

	original := strings.Repeat("2026-06-16T20:00:01Z WARN retrying request\n", 12) +
		"2026-06-16T20:00:03Z ERROR request failed permanently"
	pipeline := NewPipeline(PipelineOptions{
		Counters: tokens.NewCounterSet(),
		ProviderModel: tokens.ProviderModel{
			Provider: string(ProviderOpenAI),
			Model:    "gpt-test",
		},
		Compressor: NativeTextLogCompressor{},
	})

	result, err := pipeline.Apply([]Block{{
		ID:          "block_1",
		Provider:    ProviderOpenAI,
		ContentType: ContentLog,
		Text:        original,
		Selected:    true,
		Stability:   StabilityLive,
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if !result.Decisions[0].Applied {
		t.Fatalf("decision = %#v, want validation to apply compressed log", result.Decisions[0])
	}
	if !strings.Contains(result.Blocks[0].Text, "ERROR request failed permanently") {
		t.Fatalf("compressed text missing error signal: %q", result.Blocks[0].Text)
	}
	if result.Stats.TokenDelta <= 0 {
		t.Fatalf("token delta = %d, want positive savings", result.Stats.TokenDelta)
	}
}

func TestNativeTextLogCompressorLeavesUnsupportedTypesForValidationFallback(t *testing.T) {
	t.Parallel()

	proposal, err := NativeTextLogCompressor{}.Compress(Block{
		ID:          "block_1",
		ContentType: ContentJSON,
		Text:        `{"keep":"unchanged"}`,
	})
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}
	if proposal.Text != `{"keep":"unchanged"}` {
		t.Fatalf("proposal = %q, want unsupported content unchanged", proposal.Text)
	}
}
