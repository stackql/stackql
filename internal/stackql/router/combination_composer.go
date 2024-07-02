package router

var (
	_ combinationComposer = &standardCombinationComposer{}
)

type parameterPack map[string][]any

type combinationComposer interface {
	analyse(parameterPack) error
	getCombinations() []map[string]any
	getIsAnythingSplit() bool
}

type standardCombinationComposer struct {
	isAnythingSplit bool
	combinations    []map[string]any
}

func newCombinationComposer() combinationComposer {
	return &standardCombinationComposer{}
}

func (sc *standardCombinationComposer) getCombinations() []map[string]any {
	return sc.combinations
}

func (sc *standardCombinationComposer) getIsAnythingSplit() bool {
	return sc.isAnythingSplit
}

func (sc *standardCombinationComposer) getPi(pp parameterPack, keys []string) int {
	var pi int = 1 //nolint:revive,stylecheck // prefer explicit
	for _, k := range keys {
		pi *= len(pp[k])
	}
	return pi
}

func (sc *standardCombinationComposer) getRotation(x int, piMMinusOne int) int {
	return x / piMMinusOne // this is floor division for golang int type
}

func (sc *standardCombinationComposer) getModularOrdinal(x int, piM int, piMMinusOne int) int {
	// demi rotation modulo floor quotient of pi of M and pi of M exlusive of the current parameter
	return sc.getRotation(x, piMMinusOne) % (piM / piMMinusOne)
}

func (sc *standardCombinationComposer) analyse(
	pp parameterPack,
) error {
	var keyOrdering []string
	for k := range pp {
		keyOrdering = append(keyOrdering, k)
	}
	totalCombinationCount := sc.getPi(pp, keyOrdering)
	if totalCombinationCount > 1 {
		sc.isAnythingSplit = true
	}
	var modularOrdinals []map[string]int
	for x := 0; x < totalCombinationCount; x++ {
		modularOrdinalIter := make(map[string]int)
		for i, key := range keyOrdering {
			modularOrdinalIter[key] = sc.getModularOrdinal(x, sc.getPi(pp, keyOrdering[:i+1]), sc.getPi(pp, keyOrdering[:i]))
		}
		modularOrdinals = append(modularOrdinals, modularOrdinalIter)
	}
	for _, mo := range modularOrdinals {
		combination := make(map[string]any)
		for k, v := range mo {
			combination[k] = pp[k][v]
		}
		sc.combinations = append(sc.combinations, combination)
	}
	return nil
}
