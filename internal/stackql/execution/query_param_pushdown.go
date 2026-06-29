package execution

import (
	"github.com/stackql/any-sdk/public/formulation"
)

// pushdownArmouryGenerator carries out a push-down plan: it decorates a
// BaseArmouryGenerator, merging the (planning-computed) query parameters into every
// request param's query string. The decision of which params to push is made earlier,
// in the analysis phase (internal/stackql/pushdown); the executor only applies it.
type pushdownArmouryGenerator struct {
	prior       formulation.BaseArmouryGenerator
	queryParams map[string]string
}

// newPushdownArmouryGenerator wraps prior with the supplied push-down query params,
// returning prior unchanged when there is nothing to push.
func newPushdownArmouryGenerator(
	prior formulation.BaseArmouryGenerator,
	queryParams map[string]string,
) formulation.BaseArmouryGenerator {
	if len(queryParams) == 0 {
		return prior
	}
	return &pushdownArmouryGenerator{prior: prior, queryParams: queryParams}
}

func (g *pushdownArmouryGenerator) GetHTTPArmoury() (formulation.HTTPArmoury, error) {
	armoury, err := g.prior.GetHTTPArmoury()
	if err != nil || len(g.queryParams) == 0 {
		return armoury, err
	}
	params := armoury.GetRequestParams()
	for i, p := range params {
		param := p
		q := param.GetQuery()
		for k, v := range g.queryParams {
			q.Set(k, v)
		}
		param.SetRawQuery(q.Encode())
		params[i] = param
	}
	armoury.SetRequestParams(params)
	return armoury, nil
}
