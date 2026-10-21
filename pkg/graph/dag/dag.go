/*
 *     Copyright 2022 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//go:generate mockgen -destination mocks/dag_mock.go -source dag.go -package mocks

package dag

import (
	"errors"
	"maps"
	"math/rand"
	"sync"

	"d7y.io/dragonfly/v2/pkg/container/set"
)

var (
	// ErrVertexNotFound represents vertex not found.
	ErrVertexNotFound = errors.New("vertex not found")

	// ErrVertexInvalid represents vertex invalid.
	ErrVertexInvalid = errors.New("vertex invalid")

	// ErrVertexAlreadyExists represents vertex already exists.
	ErrVertexAlreadyExists = errors.New("vertex already exists")

	// ErrParnetAlreadyExists represents parent of vertex already exists.
	ErrParnetAlreadyExists = errors.New("parent of vertex already exists")

	// ErrChildAlreadyExists represents child of vertex already exists.
	ErrChildAlreadyExists = errors.New("child of vertex already exists")

	// ErrCycleBetweenVertices represents cycle between vertices.
	ErrCycleBetweenVertices = errors.New("cycle between vertices")
)

// DAG is the interface used for directed acyclic graph.
type DAG[T comparable] interface {
	// AddVertex adds vertex to graph.
	AddVertex(id string, value T) error

	// DeleteVertex deletes vertex graph.
	DeleteVertex(id string)

	// GetVertex gets vertex from graph.
	GetVertex(id string) (*Vertex[T], error)

	// GetVertices returns map of vertices.
	GetVertices() map[string]*Vertex[T]

	// Range calls f for each vertex in the graph without copying vertices.
	// If f returns false, range stops the iteration.
	Range(f func(id string, vertex *Vertex[T]) bool)

	// GetRandomVertices returns random map of vertices.
	GetRandomVertices(n uint) []*Vertex[T]

	// GetSourceVertices returns source vertices.
	GetSourceVertices() []*Vertex[T]

	// GetSinkVertices returns sink vertices.
	GetSinkVertices() []*Vertex[T]

	// VertexCount returns count of vertices.
	VertexCount() uint64

	// AddEdge adds edge between two vertices.
	AddEdge(fromVertexID, toVertexID string) error

	// AddEdges adds edges from each of the given from-vertices to the to-vertex.
	AddEdges(fromVertexIDs []string, toVertexID string) map[string]struct{}

	// DeleteEdge deletes edge between two vertices.
	DeleteEdge(fromVertexID, toVertexID string) error

	// CanAddEdge finds whether there are circles through depth-first search.
	CanAddEdge(fromVertexID, toVertexID string) bool

	// CanAddEdges reports which of the given from-vertices can currently have an
	// edge added to the to-vertex.
	CanAddEdges(fromVertexIDs []string, toVertexID string) map[string]struct{}

	// DeleteVertexInEdges deletes inedges of vertex.
	DeleteVertexInEdges(id string) error

	// DeleteVertexOutEdges deletes outedges of vertex.
	DeleteVertexOutEdges(id string) error
}

// dag provides directed acyclic graph function.
type dag[T comparable] struct {
	mu       sync.RWMutex
	vertices map[string]*Vertex[T]
}

// New returns a new DAG interface.
func NewDAG[T comparable]() DAG[T] {
	return &dag[T]{
		vertices: make(map[string]*Vertex[T]),
	}
}

// AddVertex adds vertex to graph.
func (d *dag[T]) AddVertex(id string, value T) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.vertices[id]; ok {
		return ErrVertexAlreadyExists
	}

	d.vertices[id] = NewVertex(id, value)
	return nil
}

// DeleteVertex deletes vertex graph.
func (d *dag[T]) DeleteVertex(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	vertex, ok := d.vertices[id]
	if !ok {
		return
	}

	for _, parent := range vertex.Parents.Values() {
		parent.Children.Delete(vertex)
	}

	for _, child := range vertex.Children.Values() {
		child.Parents.Delete(vertex)
	}

	delete(d.vertices, id)
}

// GetVertex gets vertex from graph.
func (d *dag[T]) GetVertex(id string) (*Vertex[T], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.getVertex(id)
}

// getVertex gets vertex from graph without locking, the caller must hold the lock.
func (d *dag[T]) getVertex(id string) (*Vertex[T], error) {
	vertex, ok := d.vertices[id]
	if !ok {
		return nil, ErrVertexNotFound
	}

	return vertex, nil
}

// GetVertices returns map of vertices.
func (d *dag[T]) GetVertices() map[string]*Vertex[T] {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return maps.Clone(d.vertices)
}

// Range calls f for each vertex in the graph without copying vertices.
// If f returns false, range stops the iteration.
func (d *dag[T]) Range(f func(id string, vertex *Vertex[T]) bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for id, vertex := range d.vertices {
		if !f(id, vertex) {
			return
		}
	}
}

// GetRandomVertices returns random map of vertices.
func (d *dag[T]) GetRandomVertices(n uint) []*Vertex[T] {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if n == 0 {
		return nil
	}

	// Use reservoir sampling so every vertex has an equal chance of being selected
	// without allocating a slice for the entire graph.
	randomVertices := make([]*Vertex[T], 0, n)
	count := 0
	for _, vertex := range d.vertices {
		if uint(len(randomVertices)) < n {
			randomVertices = append(randomVertices, vertex)
		} else {
			index := rand.Intn(count + 1)
			if index < int(n) {
				randomVertices[index] = vertex
			}
		}
		count++
	}

	return randomVertices
}

// VertexCount returns count of vertices.
func (d *dag[T]) VertexCount() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return uint64(len(d.vertices))
}

// AddEdge adds edge between two vertices.
func (d *dag[T]) AddEdge(fromVertexID, toVertexID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if fromVertexID == toVertexID {
		return ErrCycleBetweenVertices
	}

	fromVertex, err := d.getVertex(fromVertexID)
	if err != nil {
		return err
	}

	toVertex, err := d.getVertex(toVertexID)
	if err != nil {
		return err
	}

	for _, child := range fromVertex.Children.Values() {
		if child.ID == toVertexID {
			return ErrCycleBetweenVertices
		}
	}

	if d.depthFirstSearch(toVertexID, fromVertexID) {
		return ErrCycleBetweenVertices
	}

	if ok := fromVertex.Children.Add(toVertex); !ok {
		return ErrChildAlreadyExists
	}

	if ok := toVertex.Parents.Add(fromVertex); !ok {
		return ErrParnetAlreadyExists
	}

	return nil
}

// AddEdges adds edges from each of the given from-vertices to the to-vertex.
func (d *dag[T]) AddEdges(fromVertexIDs []string, toVertexID string) map[string]struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()

	toVertex, err := d.getVertex(toVertexID)
	if err != nil {
		return nil
	}

	successors := make(map[string]struct{})
	d.search(toVertexID, successors)
	added := make(map[string]struct{}, len(fromVertexIDs))
	for _, fromVertexID := range fromVertexIDs {
		if fromVertexID == toVertexID {
			continue
		}

		fromVertex, err := d.getVertex(fromVertexID)
		if err != nil {
			continue
		}

		if _, ok := successors[fromVertexID]; ok {
			continue
		}

		if ok := fromVertex.Children.Add(toVertex); !ok {
			continue
		}

		if ok := toVertex.Parents.Add(fromVertex); !ok {
			fromVertex.Children.Delete(toVertex)
			continue
		}

		added[fromVertexID] = struct{}{}
	}

	return added
}

// DeleteEdge deletes edge between two vertices.
func (d *dag[T]) DeleteEdge(fromVertexID, toVertexID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	fromVertex, err := d.getVertex(fromVertexID)
	if err != nil {
		return err
	}

	toVertex, err := d.getVertex(toVertexID)
	if err != nil {
		return err
	}

	fromVertex.Children.Delete(toVertex)
	toVertex.Parents.Delete(fromVertex)
	return nil
}

// CanAddEdge finds whether there are circles through depth-first search.
func (d *dag[T]) CanAddEdge(fromVertexID, toVertexID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if fromVertexID == toVertexID {
		return false
	}

	fromVertex, err := d.getVertex(fromVertexID)
	if err != nil {
		return false
	}

	if _, err := d.getVertex(toVertexID); err != nil {
		return false
	}

	for _, child := range fromVertex.Children.Values() {
		if child.ID == toVertexID {
			return false
		}
	}

	if d.depthFirstSearch(toVertexID, fromVertexID) {
		return false
	}

	return true
}

// CanAddEdges reports which of the given from-vertices can currently have an
// edge added to the to-vertex.
func (d *dag[T]) CanAddEdges(fromVertexIDs []string, toVertexID string) map[string]struct{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	toVertex, err := d.getVertex(toVertexID)
	if err != nil {
		return nil
	}

	successors := make(map[string]struct{})
	d.search(toVertexID, successors)

	addable := make(map[string]struct{}, len(fromVertexIDs))
	for _, fromVertexID := range fromVertexIDs {
		if fromVertexID == toVertexID {
			continue
		}

		fromVertex, err := d.getVertex(fromVertexID)
		if err != nil {
			continue
		}

		// Edge already exists.
		if toVertex.Parents.Contains(fromVertex) {
			continue
		}

		// Adding the edge would create a cycle.
		if _, ok := successors[fromVertexID]; ok {
			continue
		}

		addable[fromVertexID] = struct{}{}
	}

	return addable
}

// DeleteVertexInEdges deletes inedges of vertex.
func (d *dag[T]) DeleteVertexInEdges(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	vertex, err := d.getVertex(id)
	if err != nil {
		return err
	}

	for _, parent := range vertex.Parents.Values() {
		parent.Children.Delete(vertex)
	}

	vertex.Parents = set.NewSafeSet[*Vertex[T]]()
	return nil
}

// DeleteVertexOutEdges deletes outedges of vertex.
func (d *dag[T]) DeleteVertexOutEdges(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	vertex, err := d.getVertex(id)
	if err != nil {
		return err
	}

	for _, child := range vertex.Children.Values() {
		child.Parents.Delete(vertex)
	}

	vertex.Children = set.NewSafeSet[*Vertex[T]]()
	return nil
}

// GetSourceVertices returns source vertices.
func (d *dag[T]) GetSourceVertices() []*Vertex[T] {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var sourceVertices []*Vertex[T]
	for _, vertex := range d.vertices {
		if vertex.InDegree() == 0 {
			sourceVertices = append(sourceVertices, vertex)
		}
	}

	return sourceVertices
}

// GetSinkVertices returns sink vertices.
func (d *dag[T]) GetSinkVertices() []*Vertex[T] {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var sinkVertices []*Vertex[T]
	for _, vertex := range d.vertices {
		if vertex.OutDegree() == 0 {
			sinkVertices = append(sinkVertices, vertex)
		}
	}

	return sinkVertices
}

// depthFirstSearch is a depth-first search of the directed acyclic graph.
func (d *dag[T]) depthFirstSearch(fromVertexID, toVertexID string) bool {
	successors := make(map[string]struct{})
	d.search(fromVertexID, successors)
	_, ok := successors[toVertexID]
	return ok
}

// search finds successors of vertex iteratively, the caller must hold the lock.
func (d *dag[T]) search(vertexID string, successors map[string]struct{}) {
	vertex, err := d.getVertex(vertexID)
	if err != nil {
		return
	}

	stack := []*Vertex[T]{vertex}
	for len(stack) > 0 {
		vertex = stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, child := range vertex.Children.Values() {
			if _, ok := successors[child.ID]; !ok {
				successors[child.ID] = struct{}{}
				stack = append(stack, child)
			}
		}
	}
}
