package compress

import (
	"strings"
	"testing"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/tokens"
)

func TestPipelineAppliesSelectedProposalWithTokenSavings(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(PipelineOptions{
		Counters: tokens.NewCounterSet(),
		ProviderModel: tokens.ProviderModel{
			Provider: string(ProviderOpenAI),
			Model:    "gpt-test",
		},
		Compressor: CompressorFunc(func(block Block) (Proposal, error) {
			if block.ID != "block_2" {
				t.Fatalf("compressed block %q, want selected block_2", block.ID)
			}
			return Proposal{Text: "short result"}, nil
		}),
	})

	result, err := pipeline.Apply([]Block{
		{ID: "block_1", Provider: ProviderOpenAI, Text: "stable prefix", Selected: false, Stability: StabilityStable},
		{ID: "block_2", Provider: ProviderOpenAI, Text: strings.Repeat("large tool output ", 20), Selected: true, Stability: StabilityLive},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(result.Blocks))
	}
	if result.Blocks[0].Text != "stable prefix" {
		t.Fatalf("protected block text = %q, want original", result.Blocks[0].Text)
	}
	if result.Blocks[1].Text != "short result" {
		t.Fatalf("selected block text = %q, want proposal", result.Blocks[1].Text)
	}
	if result.Stats.Applied != 1 || result.Stats.Skipped != 1 || result.Stats.TokenDelta <= 0 {
		t.Fatalf("stats = %#v, want one applied, one skipped, positive delta", result.Stats)
	}
	if !result.Decisions[1].Applied || result.Decisions[1].Reason != DecisionApplied {
		t.Fatalf("selected decision = %#v, want applied", result.Decisions[1])
	}
}

func TestPipelineFallsBackWhenProposalHasNoSavingsOrIsLarger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		proposed string
	}{
		{name: "same size", original: "alpha beta gamma", proposed: "alpha beta gamma"},
		{name: "larger", original: "alpha beta", proposed: "alpha beta gamma delta epsilon"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pipeline := NewPipeline(PipelineOptions{
				Counters: tokens.NewCounterSet(),
				ProviderModel: tokens.ProviderModel{
					Provider: string(ProviderOpenAI),
					Model:    "gpt-test",
				},
				Compressor: CompressorFunc(func(Block) (Proposal, error) {
					return Proposal{Text: tt.proposed}, nil
				}),
			})

			result, err := pipeline.Apply([]Block{{
				ID:        "block_1",
				Provider:  ProviderOpenAI,
				Text:      tt.original,
				Selected:  true,
				Stability: StabilityLive,
			}})
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			if result.Blocks[0].Text != tt.original {
				t.Fatalf("block text = %q, want original %q", result.Blocks[0].Text, tt.original)
			}
			if result.Decisions[0].Applied {
				t.Fatalf("decision = %#v, want fallback", result.Decisions[0])
			}
			if result.Decisions[0].Reason != DecisionNoSavings {
				t.Fatalf("reason = %q, want %q", result.Decisions[0].Reason, DecisionNoSavings)
			}
		})
	}
}

func TestPipelineFallsBackWhenProposalIsInvalidUTF8(t *testing.T) {
	t.Parallel()

	original := "valid original text with many repeated words"
	pipeline := NewPipeline(PipelineOptions{
		Counters: tokens.NewCounterSet(),
		ProviderModel: tokens.ProviderModel{
			Provider: string(ProviderOpenAI),
			Model:    "gpt-test",
		},
		Compressor: CompressorFunc(func(Block) (Proposal, error) {
			return Proposal{Bytes: []byte{0xff, 0xfe, 0xfd}}, nil
		}),
	})

	result, err := pipeline.Apply([]Block{{
		ID:        "block_1",
		Provider:  ProviderOpenAI,
		Text:      original,
		Selected:  true,
		Stability: StabilityLive,
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Blocks[0].Text != original {
		t.Fatalf("block text = %q, want original", result.Blocks[0].Text)
	}
	if result.Decisions[0].Reason != DecisionInvalidUTF8 {
		t.Fatalf("reason = %q, want %q", result.Decisions[0].Reason, DecisionInvalidUTF8)
	}
}

func TestPipelineDoesNotCompressProtectedFields(t *testing.T) {
	t.Parallel()

	called := false
	pipeline := NewPipeline(PipelineOptions{
		Counters: tokens.NewCounterSet(),
		ProviderModel: tokens.ProviderModel{
			Provider: string(ProviderOpenAI),
			Model:    "gpt-test",
		},
		Compressor: CompressorFunc(func(Block) (Proposal, error) {
			called = true
			return Proposal{Text: "mutated"}, nil
		}),
	})

	result, err := pipeline.Apply([]Block{{
		ID:        "block_1",
		Provider:  ProviderOpenAI,
		Text:      "system prompt must not move",
		Selected:  false,
		Stability: StabilityStable,
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if called {
		t.Fatal("compressor called for protected block")
	}
	if result.Blocks[0].Text != "system prompt must not move" {
		t.Fatalf("block text = %q, want protected original", result.Blocks[0].Text)
	}
	if result.Decisions[0].Reason != DecisionNotSelected {
		t.Fatalf("reason = %q, want %q", result.Decisions[0].Reason, DecisionNotSelected)
	}
}

func TestPipelineTriesCompressorsInOrderUntilProposalSavesTokens(t *testing.T) {
	t.Parallel()

	var calls []string
	original := strings.Repeat("native cannot reduce this block ", 12)
	pipeline := NewPipeline(PipelineOptions{
		Counters: tokens.NewCounterSet(),
		ProviderModel: tokens.ProviderModel{
			Provider: string(ProviderOpenAI),
			Model:    "gpt-test",
		},
		Compressors: []Compressor{
			CompressorFunc(func(Block) (Proposal, error) {
				calls = append(calls, "native")
				return Proposal{Text: original}, nil
			}),
			CompressorFunc(func(Block) (Proposal, error) {
				calls = append(calls, "wasm")
				return Proposal{Text: "short plugin result"}, nil
			}),
		},
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

	if got := strings.Join(calls, ","); got != "native,wasm" {
		t.Fatalf("calls = %q, want native,wasm", got)
	}
	if result.Blocks[0].Text != "short plugin result" {
		t.Fatalf("block text = %q, want WASM proposal", result.Blocks[0].Text)
	}
	if !result.Decisions[0].Applied || result.Decisions[0].Reason != DecisionApplied {
		t.Fatalf("decision = %#v, want applied WASM proposal", result.Decisions[0])
	}
}

func TestPipelineStopsBeforeWASMWhenNativeProposalSavesTokens(t *testing.T) {
	t.Parallel()

	wasmCalled := false
	pipeline := NewPipeline(PipelineOptions{
		Counters: tokens.NewCounterSet(),
		ProviderModel: tokens.ProviderModel{
			Provider: string(ProviderOpenAI),
			Model:    "gpt-test",
		},
		Compressors: []Compressor{
			CompressorFunc(func(Block) (Proposal, error) {
				return Proposal{Text: "native short"}, nil
			}),
			CompressorFunc(func(Block) (Proposal, error) {
				wasmCalled = true
				return Proposal{Text: "wasm short"}, nil
			}),
		},
	})

	result, err := pipeline.Apply([]Block{{
		ID:          "block_1",
		Provider:    ProviderOpenAI,
		ContentType: ContentLog,
		Text:        strings.Repeat("native handles this block ", 12),
		Selected:    true,
		Stability:   StabilityLive,
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if wasmCalled {
		t.Fatal("WASM compressor called after native proposal already saved tokens")
	}
	if result.Blocks[0].Text != "native short" {
		t.Fatalf("block text = %q, want native proposal", result.Blocks[0].Text)
	}
}
