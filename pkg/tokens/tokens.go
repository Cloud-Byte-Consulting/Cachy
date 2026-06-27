// Package tokens exposes Cachy's token-counting abstractions for applications
// that want to reuse compression validation logic without running the proxy.
package tokens

import internaltokens "github.com/cloud-byte-consulting/cachy/internal/tokens"

const MethodEstimated = internaltokens.MethodEstimated

type ProviderModel = internaltokens.ProviderModel
type Count = internaltokens.Count
type Savings = internaltokens.Savings
type Counter = internaltokens.Counter
type CounterFunc = internaltokens.CounterFunc
type CounterSet = internaltokens.CounterSet

func NewCounterSet() *CounterSet {
	return internaltokens.NewCounterSet()
}

func Estimate(text string) Count {
	return internaltokens.Estimate(text)
}
