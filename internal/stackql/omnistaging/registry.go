package omnistaging

import (
	"context"
	"fmt"
	"sync"
)

// counter allocates collision-free query IDs.
type counter interface {
	Next(ctx context.Context) (QueryID, error)
}

// sqlCounter delegates allocation to the backend, so IDs are unique across
// every process sharing that backend.
type sqlCounter struct {
	db      Executor
	dialect Dialect
}

func newSQLCounter(db Executor, dialect Dialect) counter {
	return &sqlCounter{db: db, dialect: dialect}
}

func (c *sqlCounter) Next(ctx context.Context) (QueryID, error) {
	var value int64
	if err := c.db.QueryRowContext(ctx, c.dialect.NextQueryIDStatement()).Scan(&value); err != nil {
		return nil, fmt.Errorf("omnistaging: cannot allocate query id: %w", err)
	}
	return newQueryID(value), nil
}

// registry is the concurrency-safe store of live queries, their parent/child
// associations and whether each owns a staged table.
type registry interface {
	Add(id QueryID, parent QueryID) error
	Contains(id QueryID) bool
	Parent(id QueryID) (QueryID, bool)
	Children(id QueryID) []QueryID
	MarkStaged(id QueryID) error
	// Subtree returns id and all descendants, descendants first.
	Subtree(id QueryID) []QueryID
	IsStaged(id QueryID) bool
	Remove(ids []QueryID)
}

type registryNode struct {
	id       QueryID
	parent   QueryID
	children []QueryID
	staged   bool
}

type standardRegistry struct {
	mu    sync.Mutex
	nodes map[int64]*registryNode
}

func newRegistry() registry {
	return &standardRegistry{nodes: make(map[int64]*registryNode)}
}

func (r *standardRegistry) Add(id QueryID, parent QueryID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.nodes[id.Value()]; ok {
		return fmt.Errorf("omnistaging: query %s already registered", id)
	}
	if parent != nil {
		parentNode, ok := r.nodes[parent.Value()]
		if !ok {
			return fmt.Errorf("omnistaging: parent query %s not registered", parent)
		}
		parentNode.children = append(parentNode.children, id)
	}
	r.nodes[id.Value()] = &registryNode{id: id, parent: parent}
	return nil
}

func (r *standardRegistry) Contains(id QueryID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.nodes[id.Value()]
	return ok
}

func (r *standardRegistry) Parent(id QueryID) (QueryID, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.nodes[id.Value()]
	if !ok || node.parent == nil {
		return nil, false
	}
	return node.parent, true
}

func (r *standardRegistry) Children(id QueryID) []QueryID {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.nodes[id.Value()]
	if !ok {
		return nil
	}
	return append([]QueryID(nil), node.children...)
}

func (r *standardRegistry) MarkStaged(id QueryID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.nodes[id.Value()]
	if !ok {
		return fmt.Errorf("omnistaging: query %s not registered", id)
	}
	node.staged = true
	return nil
}

func (r *standardRegistry) IsStaged(id QueryID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.nodes[id.Value()]
	return ok && node.staged
}

func (r *standardRegistry) Subtree(id QueryID) []QueryID {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []QueryID
	var walk func(QueryID)
	walk = func(current QueryID) {
		node, ok := r.nodes[current.Value()]
		if !ok {
			return
		}
		for _, child := range node.children {
			walk(child)
		}
		out = append(out, node.id)
	}
	walk(id)
	return out
}

func (r *standardRegistry) Remove(ids []QueryID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		node, ok := r.nodes[id.Value()]
		if !ok {
			continue
		}
		if node.parent != nil {
			if parentNode, parentOK := r.nodes[node.parent.Value()]; parentOK {
				parentNode.children = removeQueryID(parentNode.children, id)
			}
		}
		delete(r.nodes, id.Value())
	}
}

func removeQueryID(ids []QueryID, target QueryID) []QueryID {
	out := ids[:0]
	for _, id := range ids {
		if id.Value() != target.Value() {
			out = append(out, id)
		}
	}
	return out
}
