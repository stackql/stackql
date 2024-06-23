package dataflow

import (
	"fmt"
	"sync"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type Unit interface {
	iDataFlowUnit()
}

// Collection is the DAG representing.
type Collection interface {
	AddOrUpdateEdge(
		source Vertex,
		dest Vertex,
		comparisonExpr *sqlparser.ComparisonExpr,
		sourceExpr sqlparser.Expr,
		destColumn *sqlparser.ColName) error
	AddVertex(v Vertex)
	GetAllUnits() ([]Unit, error)
	GetNextID() int64
	InDegree(v Vertex) int
	OutDegree(v Vertex) int
	Sort() error
	Vertices() []Vertex
	UpsertStandardDataFlowVertex(taxonomy.AnnotationCtx, sqlparser.TableExpr) Vertex
	getVertex(annotation taxonomy.AnnotationCtx, tableExpr sqlparser.TableExpr) (Vertex, bool)
	GetSplitAnnotationContextMap() taxonomy.AnnotationCtxSplitMap
	WithSplitAnnotationContextMap(taxonomy.AnnotationCtxSplitMap) Collection
}

func NewStandardDataFlowCollection() Collection {
	return &standardDataFlowCollection{
		idMutex:                   &sync.Mutex{},
		g:                         simple.NewWeightedDirectedGraph(0.0, 0.0),
		vertices:                  make(map[Vertex]struct{}),
		verticesForTableExprs:     make(map[sqlparser.TableExpr]struct{}),
		splitAnnotationContextMap: taxonomy.NewAnnotationCtxSplitMap(),
	}
}

// Upsert a vertex into the collection.
// If the vertex already exists, it is returned.
// Otherwise, a new vertex is created and returned.
// This makes subsequent handling of degenerate edges easier.
func (dc *standardDataFlowCollection) UpsertStandardDataFlowVertex(
	annotation taxonomy.AnnotationCtx,
	tableExpr sqlparser.TableExpr,
) Vertex {
	if v, ok := dc.getVertex(annotation, tableExpr); ok {
		return v
	}
	return &standardDataFlowVertex{
		annotation: annotation,
		tableExpr:  tableExpr,
		id:         dc.GetNextID(),
	}
}

func (dc *standardDataFlowCollection) getVertex(
	annotation taxonomy.AnnotationCtx,
	tableExpr sqlparser.TableExpr,
) (Vertex, bool) {
	for v := range dc.vertices {
		if v.GetAnnotation() == annotation && v.GetTableExpr() == tableExpr {
			return v, true
		}
	}
	return nil, false
}

type standardDataFlowCollection struct {
	idMutex                   *sync.Mutex
	maxID                     int64
	g                         *simple.WeightedDirectedGraph
	sorted                    []graph.Node
	orphans                   []Vertex
	weaklyConnnectedGraphs    []WeaklyConnectedComponent
	vertices                  map[Vertex]struct{}
	verticesForTableExprs     map[sqlparser.TableExpr]struct{}
	edges                     []Edge
	splitAnnotationContextMap taxonomy.AnnotationCtxSplitMap
}

func (dc *standardDataFlowCollection) GetNextID() int64 {
	defer dc.idMutex.Unlock()
	dc.idMutex.Lock()
	dc.maxID++
	return dc.maxID
}

func (dc *standardDataFlowCollection) GetSplitAnnotationContextMap() taxonomy.AnnotationCtxSplitMap {
	return dc.splitAnnotationContextMap
}

func (dc *standardDataFlowCollection) WithSplitAnnotationContextMap(m taxonomy.AnnotationCtxSplitMap) Collection {
	dc.splitAnnotationContextMap = m
	return dc
}

func (dc *standardDataFlowCollection) AddOrUpdateEdge(
	source Vertex,
	dest Vertex,
	comparisonExpr *sqlparser.ComparisonExpr,
	sourceExpr sqlparser.Expr,
	destColumn *sqlparser.ColName,
) error {
	dc.AddVertex(source)
	dc.AddVertex(dest)
	sourceID := source.ID()
	destID := dest.ID()
	existingEdge := dc.g.WeightedEdge(sourceID, destID)
	if existingEdge == nil {
		edge := newStandardDataFlowEdge(source, dest, comparisonExpr, sourceExpr, destColumn)
		dc.edges = append(dc.edges, edge)
		dc.g.SetWeightedEdge(edge)
		return nil
	}
	switch existingEdge := existingEdge.(type) {
	case Edge:
		existingEdge.AddRelation(NewStandardDataFlowRelation(comparisonExpr, destColumn, sourceExpr))
	default:
		return fmt.Errorf("cannnot accomodate data flow edge of type: '%T'", existingEdge)
	}
	return nil
}

func (dc *standardDataFlowCollection) AddVertex(v Vertex) {
	_, ok := dc.verticesForTableExprs[v.GetTableExpr()]
	if ok {
		var isNew bool
		if v.GetEquivalencyGroup() > 0 {
			_, isNew = dc.g.NodeWithID(v.ID()) // TODO: change to acceptable idion
		}
		if !isNew {
			return
		}
	}
	dc.vertices[v] = struct{}{}
	dc.verticesForTableExprs[v.GetTableExpr()] = struct{}{}
	dc.g.AddNode(v)
}

// Sort sorts the underlying graph.
// Cyclic dependencies return an error.
// This is where the DAG invariant is enforced.
func (dc *standardDataFlowCollection) Sort() error {
	var err error
	dc.sorted, err = topo.Sort(dc.g)
	if err != nil {
		return err
	}
	err = dc.optimise()
	return err
}

func (dc *standardDataFlowCollection) Vertices() []Vertex {
	var rv []Vertex
	for vert := range dc.vertices {
		rv = append(rv, vert)
	}
	return rv
}

func (dc *standardDataFlowCollection) GetAllUnits() ([]Unit, error) {
	var rv []Unit
	for _, orphan := range dc.orphans {
		rv = append(rv, orphan)
	}
	for _, component := range dc.weaklyConnnectedGraphs {
		rv = append(rv, component)
	}
	return rv, nil
}

func (dc *standardDataFlowCollection) InDegree(v Vertex) int {
	inDegree := 0
	for _, e := range dc.edges {
		if e.GetDest() == v {
			inDegree++
		}
	}
	return inDegree
}

func (dc *standardDataFlowCollection) OutDegree(v Vertex) int {
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
		case Vertex:
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
