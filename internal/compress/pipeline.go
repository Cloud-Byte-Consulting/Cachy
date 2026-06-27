package compress

import (
	"errors"
	"unicode/utf8"

	"github.com/cloud-byte-consulting/cachy/internal/tokens"
)

type DecisionReason string

const (
	DecisionApplied     DecisionReason = "applied"
	DecisionNotSelected DecisionReason = "not_selected"
	DecisionNoSavings   DecisionReason = "no_savings"
	DecisionInvalidUTF8 DecisionReason = "invalid_utf8"
	DecisionError       DecisionReason = "compressor_error"
)

type Proposal struct {
	Text  string
	Bytes []byte
}

type Compressor interface {
	Compress(Block) (Proposal, error)
}

type CompressorFunc func(Block) (Proposal, error)

func (f CompressorFunc) Compress(block Block) (Proposal, error) {
	return f(block)
}

type PipelineOptions struct {
	Compressor    Compressor
	Compressors   []Compressor
	Counters      *tokens.CounterSet
	ProviderModel tokens.ProviderModel
}

type Pipeline struct {
	compressors   []Compressor
	counters      *tokens.CounterSet
	providerModel tokens.ProviderModel
}

type PipelineResult struct {
	Blocks    []Block
	Decisions []Decision
	Stats     PipelineStats
}

type Decision struct {
	BlockID string
	Applied bool
	Reason  DecisionReason
	Savings tokens.Savings
	Err     error
}

type PipelineStats struct {
	Applied    int
	Skipped    int
	TokenDelta int
}

func NewPipeline(options PipelineOptions) *Pipeline {
	counters := options.Counters
	if counters == nil {
		counters = tokens.NewCounterSet()
	}
	compressors := append([]Compressor(nil), options.Compressors...)
	if len(compressors) == 0 && options.Compressor != nil {
		compressors = append(compressors, options.Compressor)
	}
	return &Pipeline{
		compressors:   compressors,
		counters:      counters,
		providerModel: options.ProviderModel,
	}
}

func (p *Pipeline) Apply(blocks []Block) (PipelineResult, error) {
	result := PipelineResult{
		Blocks:    cloneBlocks(blocks),
		Decisions: make([]Decision, len(blocks)),
	}

	for i, block := range blocks {
		decision := Decision{BlockID: block.ID}
		if !block.Selected || block.Stability != StabilityLive {
			decision.Reason = DecisionNotSelected
			result.Stats.Skipped++
			result.Decisions[i] = decision
			continue
		}

		for _, compressor := range p.compressors {
			proposal, err := compressor.Compress(block)
			if err != nil {
				decision.Reason = DecisionError
				decision.Err = err
				continue
			}

			text, savings, reason, err := p.evaluateProposal(block, proposal)
			if err != nil {
				decision.Reason = DecisionError
				decision.Err = err
				continue
			}
			decision.Savings = savings
			if reason != DecisionApplied {
				decision.Reason = reason
				continue
			}

			result.Blocks[i].Text = text
			decision.Applied = true
			decision.Reason = DecisionApplied
			result.Stats.Applied++
			result.Stats.TokenDelta += savings.Delta
			break
		}

		if decision.Reason == "" {
			decision.Reason = DecisionError
			decision.Err = errors.New("compressor is required")
		}
		if !decision.Applied {
			result.Stats.Skipped++
		}
		result.Decisions[i] = decision
	}

	return result, nil
}

func (p *Pipeline) evaluateProposal(block Block, proposal Proposal) (string, tokens.Savings, DecisionReason, error) {
	text, ok := proposalText(proposal)
	if !ok {
		return "", tokens.Savings{}, DecisionInvalidUTF8, nil
	}

	savings, err := p.counters.Savings(p.providerModel, block.Text, text)
	if err != nil {
		return "", tokens.Savings{}, DecisionError, err
	}
	if !savings.Improved {
		return text, savings, DecisionNoSavings, nil
	}
	return text, savings, DecisionApplied, nil
}

func proposalText(proposal Proposal) (string, bool) {
	if proposal.Bytes != nil {
		if !utf8.Valid(proposal.Bytes) {
			return "", false
		}
		return string(proposal.Bytes), true
	}
	if !utf8.ValidString(proposal.Text) {
		return "", false
	}
	return proposal.Text, true
}

func cloneBlocks(blocks []Block) []Block {
	cloned := make([]Block, len(blocks))
	copy(cloned, blocks)
	return cloned
}
