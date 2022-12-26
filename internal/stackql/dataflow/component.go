package dataflow

import (
	"fmt"

	"gonum.org/v1/gonum/graph"
)

type DataFlowWeaklyConnectedComponent interface {
	DataFlowUnit
	AddEdge(DataFlowEdge)
	Analyze() error
	GetEdges() ([]DataFlowEdge, error)
	GetOrderedNodes() ([]DataFlowVertex, error)
	PushBack(DataFlowVertex)
}

type standardDataFlowWeaklyConnectedComponent struct {
	idsVisited   map[int64]struct{}
	collection   *standardDataFlowCollection
	root         graph.Node
	orderedNodes []graph.Node
	edges        []graph.Edge
}

func NewStandardDataFlowWeaklyConnectedComponent(
	collection *standardDataFlowCollection,
	root graph.Node,
) DataFlowWeaklyConnectedComponent {
	return &standardDataFlowWeaklyConnectedComponent{
		collection: collection,
		root:       root,
		idsVisited: map[int64]struct{}{
			root.ID(): {},
		},
		orderedNodes: []graph.Node{
			root,
		},
	}
}

func (wc *standardDataFlowWeaklyConnectedComponent) GetOrderedNodes() ([]DataFlowVertex, error) {
	var rv []DataFlowVertex
	for _, n := range wc.orderedNodes {
		switch n := n.(type) {
		case DataFlowVertex:
			rv = append(rv, n)
		default:
			return nil, fmt.Errorf("data flow error: weakly connected components cannot accomodate nodes of type '%T'", n)
		}
	}
	return rv, nil
}

func (wc *standardDataFlowWeaklyConnectedComponent) GetEdges() ([]DataFlowEdge, error) {
	var rv []DataFlowEdge
	for _, n := range wc.edges {
		switch n := n.(type) {
		case DataFlowEdge:
			rv = append(rv, n)
		default:
			return nil, fmt.Errorf("data flow error: weakly connected components cannot accomodate edges of type '%T'", n)
		}
	}
	return rv, nil
}

func (wc *standardDataFlowWeaklyConnectedComponent) Analyze() error {
	// This algorithm underlying this analyisis
	// is defective; or would be were it not halted early.
	// TODO: Upgrade this algorithm and make limits configurable
	// and document the algorithm.
	for _, node := range wc.collection.sorted {
		incidentNodes := wc.collection.g.To(node.ID())
		wc.idsVisited[node.ID()] = struct{}{}
		if incidentNodes.Len() > 1 {
			return fmt.Errorf("data flow: too complex for now; %d dependencies detected when max allowed = 1", incidentNodes.Len())
		}
		for {
			itemPresent := incidentNodes.Next()
			if !itemPresent {
				break
			}
			fromNode := incidentNodes.Node()
			_, ok := wc.idsVisited[fromNode.ID()]
			if ok {
				wc.orderedNodes = append(wc.orderedNodes, node)
				wc.idsVisited[node.ID()] = struct{}{}
				incidentEdge := wc.collection.g.WeightedEdge(fromNode.ID(), node.ID())
				if incidentEdge == nil {
					return fmt.Errorf("found nil edge in data flow graph")
				}
				wc.edges = append(wc.edges, incidentEdge)
			} else {
				// TODO: improve error, or obviate with superior algorithm
				return fmt.Errorf("data flow: error: complexity not yet supported; only single data flow dependencies allowed at present")
			}
		}
	}
	return nil
}

func (wc *standardDataFlowWeaklyConnectedComponent) AddEdge(e DataFlowEdge) {
	wc.edges = append(wc.edges, e)
}

func (wc *standardDataFlowWeaklyConnectedComponent) PushBack(v DataFlowVertex) {
	wc.orderedNodes = append(wc.orderedNodes, v)
}

func (wc *standardDataFlowWeaklyConnectedComponent) iDataFlowUnit() {}
