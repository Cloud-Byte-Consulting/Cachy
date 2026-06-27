package tokens

import "testing"

func TestCounterSetUsesExactCounterWhenRegistered(t *testing.T) {
	t.Parallel()

	counters := NewCounterSet()
	counters.RegisterExact("openai", "gpt-test", CounterFunc(func(text string) (Count, error) {
		return Count{Tokens: 7, Exact: true, Method: "fixture"}, nil
	}))

	count, err := counters.Count(ProviderModel{Provider: "openai", Model: "gpt-test"}, "hello world")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count.Tokens != 7 || !count.Exact || count.Method != "fixture" {
		t.Fatalf("count = %#v, want exact fixture count", count)
	}
}

func TestCounterSetFallsBackToEstimatorWhenExactUnavailable(t *testing.T) {
	t.Parallel()

	counters := NewCounterSet()
	count, err := counters.Count(ProviderModel{Provider: "openai", Model: "unknown"}, "hello, world")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count.Tokens <= 0 {
		t.Fatalf("tokens = %d, want positive fallback estimate", count.Tokens)
	}
	if count.Exact {
		t.Fatal("fallback count marked exact")
	}
	if count.Method != MethodEstimated {
		t.Fatalf("method = %q, want %q", count.Method, MethodEstimated)
	}
}

func TestEstimateIsDeterministicAndConservativeForCommonText(t *testing.T) {
	t.Parallel()

	estimate := Estimate("hello world")
	if estimate.Tokens < 2 {
		t.Fatalf("estimate = %d, want at least one token per word for common text", estimate.Tokens)
	}
	if again := Estimate("hello world"); again != estimate {
		t.Fatalf("estimate not deterministic: %#v != %#v", again, estimate)
	}
	if empty := Estimate(""); empty.Tokens != 0 {
		t.Fatalf("empty estimate = %d, want 0", empty.Tokens)
	}
}

func TestSavingsUsesCountsAndRequiresPositiveImprovement(t *testing.T) {
	t.Parallel()

	counters := NewCounterSet()
	original := "alpha beta gamma delta epsilon zeta eta theta"
	proposed := "alpha beta"

	result, err := counters.Savings(ProviderModel{Provider: "local", Model: "estimator"}, original, proposed)
	if err != nil {
		t.Fatalf("Savings() error = %v", err)
	}
	if result.Original.Tokens <= result.Proposed.Tokens {
		t.Fatalf("result = %#v, want original larger than proposed", result)
	}
	if !result.Improved {
		t.Fatalf("result = %#v, want improved", result)
	}

	noSavings, err := counters.Savings(ProviderModel{Provider: "local", Model: "estimator"}, proposed, original)
	if err != nil {
		t.Fatalf("Savings() error = %v", err)
	}
	if noSavings.Improved {
		t.Fatalf("result = %#v, want not improved", noSavings)
	}
}
