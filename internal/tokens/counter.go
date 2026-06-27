package tokens

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MethodEstimated = "estimated"

type ProviderModel struct {
	Provider string
	Model    string
}

type Count struct {
	Tokens int
	Exact  bool
	Method string
}

type Savings struct {
	Original Count
	Proposed Count
	Delta    int
	Improved bool
}

type Counter interface {
	Count(text string) (Count, error)
}

type CounterFunc func(text string) (Count, error)

func (f CounterFunc) Count(text string) (Count, error) {
	return f(text)
}

type CounterSet struct {
	exact map[ProviderModel]Counter
}

func NewCounterSet() *CounterSet {
	return &CounterSet{exact: map[ProviderModel]Counter{}}
}

func (s *CounterSet) RegisterExact(provider, model string, counter Counter) {
	if s == nil || counter == nil {
		return
	}
	s.exact[ProviderModel{Provider: provider, Model: model}] = counter
}

func (s *CounterSet) Count(providerModel ProviderModel, text string) (Count, error) {
	if s != nil {
		if counter, ok := s.exact[providerModel]; ok {
			count, err := counter.Count(text)
			if err != nil {
				return Count{}, err
			}
			count.Exact = true
			if count.Method == "" {
				count.Method = providerModel.Provider + "/" + providerModel.Model
			}
			return count, nil
		}
	}
	return Estimate(text), nil
}

func (s *CounterSet) Savings(providerModel ProviderModel, original, proposed string) (Savings, error) {
	originalCount, err := s.Count(providerModel, original)
	if err != nil {
		return Savings{}, err
	}
	proposedCount, err := s.Count(providerModel, proposed)
	if err != nil {
		return Savings{}, err
	}
	delta := originalCount.Tokens - proposedCount.Tokens
	return Savings{
		Original: originalCount,
		Proposed: proposedCount,
		Delta:    delta,
		Improved: delta > 0,
	}, nil
}

func Estimate(text string) Count {
	if text == "" {
		return Count{Method: MethodEstimated}
	}

	runes := utf8.RuneCountInString(text)
	words := countWords(text)
	nonSpace := countNonSpace(text)
	punctuation := countPunctuation(text)

	byChars := int(math.Ceil(float64(nonSpace) / 4.0))
	tokens := max(words+punctuation, byChars)
	tokens = max(tokens, int(math.Ceil(float64(runes)/8.0)))
	return Count{Tokens: tokens, Exact: false, Method: MethodEstimated}
}

func countWords(text string) int {
	return len(strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	}))
}

func countNonSpace(text string) int {
	count := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func countPunctuation(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			count++
		}
	}
	return count
}
