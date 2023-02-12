package dataflow

import (
	"fmt"
	"sync"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/logging"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type DataFlowUnit interface {
	iDataFlowUnit()
}

// DataFlowCollection is the DAG representing
type DataFlowCollection interface {
	AddOrUpdateEdge(source DataFlowVertex, dest DataFlowVertex, comparisonExpr *sqlparser.ComparisonExpr, sourceExpr sqlparser.Expr, destColumn *sqlparser.ColName) error
	AddVertex(v DataFlowVertex)
	GetAllUnits() ([]DataFlowUnit, error)
	GetNextID() int64
	InDegree(v DataFlowVertex) int
	OutDegree(v DataFlowVertex) int
	Sort() error
	Vertices() []DataFlowVertex
}

func NewStandardDataFlowCollection() DataFlowCollection {
	return &standardDataFlowCollection{
		idMutex:               &sync.Mutex{},
		g:                     simple.NewWeightedDirectedGraph(0.0, 0.0),
		vertices:              make(map[DataFlowVertex]struct{}),
		verticesForTableExprs: make(map[sqlparser.TableExpr]struct{}),
	}
}

type standardDataFlowCollection struct {
	idMutex                *sync.Mutex
	maxId                  int64
	g                      *simple.WeightedDirectedGraph
	sorted                 []graph.Node
	orphans                []DataFlowVertex
	weaklyConnnectedGraphs []DataFlowWeaklyConnectedComponent
	vertices               map[DataFlowVertex]struct{}
	verticesForTableExprs  map[sqlparser.TableExpr]struct{}
	edges                  []DataFlowEdge
}

func (dc *standardDataFlowCollection) GetNextID() int64 {
	defer dc.idMutex.Unlock()
	dc.idMutex.Lock()
	dc.maxId++
	return dc.maxId
}

func (dc *standardDataFlowCollection) AddOrUpdateEdgeOld(e DataFlowEdge) error {
	dc.AddVertex(e.GetSource())
	dc.AddVertex(e.GetDest())
	dc.edges = append(dc.edges, e)
	dc.g.SetWeightedEdge(e)
	return nil
}

func (dc *standardDataFlowCollection) AddOrUpdateEdge(
	source DataFlowVertex,
	dest DataFlowVertex,
	comparisonExpr *sqlparser.ComparisonExpr,
	sourceExpr sqlparser.Expr,
	destColumn *sqlparser.ColName,
) error {
	dc.AddVertex(source)
	dc.AddVertex(dest)
	existingEdge := dc.g.WeightedEdge(source.ID(), dest.ID())
	if existingEdge == nil {
		edge := NewStandardDataFlowEdge(source, dest, comparisonExpr, sourceExpr, destColumn)
		dc.edges = append(dc.edges, edge)
		dc.g.SetWeightedEdge(edge)
		return nil
	}
	switch existingEdge := existingEdge.(type) {
	case DataFlowEdge:
		existingEdge.AddRelation(NewStandardDataFlowRelation(comparisonExpr, destColumn, sourceExpr))
	default:
		return fmt.Errorf("cannnot accomodate data flow edge of type: '%T'", existingEdge)
	}
	return nil
}

func (dc *standardDataFlowCollection) AddVertex(v DataFlowVertex) {
	_, ok := dc.verticesForTableExprs[v.GetTableExpr()]
	if ok {
		return
	}
	dc.vertices[v] = struct{}{}
	dc.verticesForTableExprs[v.GetTableExpr()] = struct{}{}
	dc.g.AddNode(v)
}

func (dc *standardDataFlowCollection) Sort() error {
	var err error
	dc.sorted, err = topo.Sort(dc.g)
	if err != nil {
		return err
	}
	err = dc.optimise()
	return err
}

func (dc *standardDataFlowCollection) Vertices() []DataFlowVertex {
	var rv []DataFlowVertex
	for vert := range dc.vertices {
		rv = append(rv, vert)
	}
	return rv
}

func (dc *standardDataFlowCollection) GetAllUnits() ([]DataFlowUnit, error) {
	var rv []DataFlowUnit
	for _, orphan := range dc.orphans {
		rv = append(rv, orphan)
	}
	for _, component := range dc.weaklyConnnectedGraphs {
		rv = append(rv, component)
	}
	return rv, nil
}

func (dc *standardDataFlowCollection) InDegree(v DataFlowVertex) int {
	inDegree := 0
	for _, e := range dc.edges {
		if e.GetDest() == v {
			inDegree++
		}
	}
	return inDegree
}

func (dc *standardDataFlowCollection) OutDegree(v DataFlowVertex) int {
	outDegree := 0
	for _, e := range dc.edges {
		if e.GetSource() == v {
			outDegree++
		}
	}
	return outDegree
}

func (dc *standardDataFlowCollection) optimise() error {
	for _, node := range dc.sorted {
		switch node := node.(type) {
		case DataFlowVertex:
			logging.GetLogger().Debugf("%v\n", node)
			inDegree := dc.g.To(node.ID()).Len()
			outDegree := dc.g.From(node.ID()).Len()
			if inDegree == 0 && outDegree == 0 {
				dc.orphans = append(dc.orphans, node)
				continue
			}
			if inDegree == 0 && outDegree != 0 {
				component := NewStandardDataFlowWeaklyConnectedComponent(dc, node)
				err := component.Analyze()
				if err != nil {
					return err
				}
				dc.weaklyConnnectedGraphs = append(dc.weaklyConnnectedGraphs, component)
			}
		default:
			return fmt.Errorf("cannot accomodate dataflow element of type = '%t'", node)
		}
	}
	return nil
}
