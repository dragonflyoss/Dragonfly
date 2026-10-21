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

package dag

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockVertexEID = "bae"
	mockVertexFID = "baf"
	mockVertexGID = "bag"
	mockVertexHID = "bah"
	mockVertexIID = "bai"
)

func mockDAG(t *testing.T, ids []string, edges [][2]string) DAG[string] {
	d := NewDAG[string]()
	for _, id := range ids {
		if err := d.AddVertex(id, mockVertexValue); err != nil {
			t.Fatal(err)
		}
	}

	for _, edge := range edges {
		if err := d.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatal(err)
		}
	}

	return d
}

func TestDAG_New(t *testing.T) {
	assert := assert.New(t)
	d := NewDAG[string]()
	assert.Equal("dag[string]", reflect.TypeOf(d).Elem().Name())
}

func TestDAG_AddVertex(t *testing.T) {
	tests := []struct {
		name   string
		ids    []string
		expect func(t *testing.T, errs []error)
	}{
		{
			name: "add vertex",
			ids:  []string{mockVertexID},
			expect: func(t *testing.T, errs []error) {
				assert := assert.New(t)
				assert.Equal([]error{nil}, errs)
			},
		},
		{
			name: "vertex already exists",
			ids:  []string{mockVertexID, mockVertexID},
			expect: func(t *testing.T, errs []error) {
				assert := assert.New(t)
				assert.Equal([]error{nil, ErrVertexAlreadyExists}, errs)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDAG[string]()
			errs := make([]error, 0, len(tc.ids))
			for _, id := range tc.ids {
				errs = append(errs, d.AddVertex(id, mockVertexValue))
			}

			tc.expect(t, errs)
		})
	}
}

func TestDAG_DeleteVertex(t *testing.T) {
	tests := []struct {
		name   string
		ids    []string
		edges  [][2]string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "delete vertex",
			ids:  []string{mockVertexID},
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				_, err := d.GetVertex(mockVertexID)
				assert.ErrorIs(err, ErrVertexNotFound)
			},
		},
		{
			name:  "delete vertex with edges",
			ids:   []string{mockVertexEID, mockVertexID, mockVertexFID},
			edges: [][2]string{{mockVertexEID, mockVertexID}, {mockVertexID, mockVertexFID}},
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				_, err := d.GetVertex(mockVertexID)
				assert.ErrorIs(err, ErrVertexNotFound)

				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(0), ve.Children.Len())

				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(0), vf.Parents.Len())
			},
		},
		{
			name: "delete unknown vertex is a no-op",
			ids:  []string{mockVertexEID},
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.Equal(uint64(1), d.VertexCount())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, tc.ids, tc.edges)
			d.DeleteVertex(mockVertexID)
			tc.expect(t, d)
		})
	}
}

func TestDAG_GetVertex(t *testing.T) {
	tests := []struct {
		name   string
		ids    []string
		expect func(t *testing.T, vertex *Vertex[string], err error)
	}{
		{
			name: "get vertex",
			ids:  []string{mockVertexID},
			expect: func(t *testing.T, vertex *Vertex[string], err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Equal(mockVertexID, vertex.ID)
				assert.Equal(mockVertexValue, vertex.Value)
				assert.Equal(uint(0), vertex.Parents.Len())
				assert.Equal(uint(0), vertex.Children.Len())
			},
		},
		{
			name: "vertex not found",
			expect: func(t *testing.T, vertex *Vertex[string], err error) {
				assert := assert.New(t)
				assert.ErrorIs(err, ErrVertexNotFound)
				assert.Nil(vertex)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, tc.ids, nil)
			vertex, err := d.GetVertex(mockVertexID)
			tc.expect(t, vertex, err)
		})
	}
}

func TestDAG_VertexCount(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "get length of vertex",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddVertex(mockVertexID, mockVertexValue))
				assert.Equal(uint64(1), d.VertexCount())

				d.DeleteVertex(mockVertexID)
				assert.Equal(uint64(0), d.VertexCount())
			},
		},
		{
			name: "empty dag",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.Equal(uint64(0), d.VertexCount())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDAG[string]()
			tc.expect(t, d)
		})
	}
}

func TestDAG_GetVertices(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "get vertices",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddVertex(mockVertexID, mockVertexValue))

				vertices := d.GetVertices()
				assert.Len(vertices, 1)
				assert.Equal(mockVertexID, vertices[mockVertexID].ID)
				assert.Equal(mockVertexValue, vertices[mockVertexID].Value)

				d.DeleteVertex(mockVertexID)
				assert.Empty(d.GetVertices())
			},
		},
		{
			name: "dag is empty",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.Empty(d.GetVertices())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDAG[string]()
			tc.expect(t, d)
		})
	}
}

func TestDAG_Range(t *testing.T) {
	tests := []struct {
		name      string
		ids       []string
		stopAfter int
		expect    func(t *testing.T, d DAG[string], visited map[string]*Vertex[string])
	}{
		{
			name: "visits every vertex without copying",
			ids:  []string{mockVertexEID, mockVertexFID, mockVertexGID},
			expect: func(t *testing.T, d DAG[string], visited map[string]*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(visited, 3)
				for _, id := range []string{mockVertexEID, mockVertexFID, mockVertexGID} {
					vertex, err := d.GetVertex(id)
					assert.NoError(err)
					assert.Same(vertex, visited[id])
				}
			},
		},
		{
			name:      "stops iterating when f returns false",
			ids:       []string{mockVertexEID, mockVertexFID, mockVertexGID},
			stopAfter: 1,
			expect: func(t *testing.T, d DAG[string], visited map[string]*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(visited, 1)
			},
		},
		{
			name: "empty dag never calls f",
			expect: func(t *testing.T, d DAG[string], visited map[string]*Vertex[string]) {
				assert := assert.New(t)
				assert.Empty(visited)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, tc.ids, nil)
			visited := make(map[string]*Vertex[string])
			d.Range(func(id string, vertex *Vertex[string]) bool {
				visited[id] = vertex
				return tc.stopAfter == 0 || len(visited) < tc.stopAfter
			})

			tc.expect(t, d, visited)
		})
	}
}

func TestDAG_GetRandomVertices(t *testing.T) {
	tests := []struct {
		name   string
		ids    []string
		n      uint
		expect func(t *testing.T, vertices []*Vertex[string])
	}{
		{
			name: "zero of two",
			ids:  []string{mockVertexEID, mockVertexFID},
			n:    0,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Empty(vertices)
			},
		},
		{
			name: "one of two",
			ids:  []string{mockVertexEID, mockVertexFID},
			n:    1,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(vertices, 1)
			},
		},
		{
			name: "two of two",
			ids:  []string{mockVertexEID, mockVertexFID},
			n:    2,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(vertices, 2)
			},
		},
		{
			name: "three of two is capped",
			ids:  []string{mockVertexEID, mockVertexFID},
			n:    3,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(vertices, 2)
			},
		},
		{
			name: "four of two is capped",
			ids:  []string{mockVertexEID, mockVertexFID},
			n:    4,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(vertices, 2)
			},
		},
		{
			name: "zero of empty dag",
			ids:  nil,
			n:    0,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Empty(vertices)
			},
		},
		{
			name: "one of empty dag",
			ids:  nil,
			n:    1,
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Empty(vertices)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, tc.ids, nil)
			tc.expect(t, d.GetRandomVertices(tc.n))
		})
	}
}

func TestDAG_GetRandomVerticesSamplesAllVertices(t *testing.T) {
	d := mockDAG(t, []string{mockVertexEID, mockVertexFID, mockVertexGID, mockVertexHID, mockVertexIID}, nil)
	allowed := map[string]struct{}{
		mockVertexEID: {},
		mockVertexFID: {},
		mockVertexGID: {},
		mockVertexHID: {},
		mockVertexIID: {},
	}

	for range 100 {
		vertices := d.GetRandomVertices(2)
		assert.Len(t, vertices, 2)
		assert.NotEqual(t, vertices[0].ID, vertices[1].ID)
		_, ok := allowed[vertices[0].ID]
		assert.True(t, ok)
		_, ok = allowed[vertices[1].ID]
		assert.True(t, ok)
	}
}

func TestDAG_AddEdge(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "add edge",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddEdge(mockVertexEID, mockVertexFID))
				assert.NoError(d.AddEdge(mockVertexFID, mockVertexGID))
				assert.NoError(d.AddEdge(mockVertexFID, mockVertexHID))
				assert.NoError(d.AddEdge(mockVertexGID, mockVertexIID))
				assert.NoError(d.AddEdge(mockVertexIID, mockVertexHID))
			},
		},
		{
			name: "cycle between vertices",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.ErrorIs(d.AddEdge(mockVertexEID, mockVertexEID), ErrCycleBetweenVertices)
				assert.NoError(d.AddEdge(mockVertexEID, mockVertexFID))
				assert.ErrorIs(d.AddEdge(mockVertexFID, mockVertexEID), ErrCycleBetweenVertices)
				assert.NoError(d.AddEdge(mockVertexFID, mockVertexGID))
				assert.ErrorIs(d.AddEdge(mockVertexGID, mockVertexEID), ErrCycleBetweenVertices)
				assert.NoError(d.AddEdge(mockVertexGID, mockVertexHID))
				assert.NoError(d.AddEdge(mockVertexHID, mockVertexIID))
				assert.ErrorIs(d.AddEdge(mockVertexIID, mockVertexEID), ErrCycleBetweenVertices)
			},
		},
		{
			name: "edge already exists",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddEdge(mockVertexEID, mockVertexFID))
				assert.ErrorIs(d.AddEdge(mockVertexEID, mockVertexFID), ErrCycleBetweenVertices)

				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(1), vf.Parents.Len())
			},
		},
		{
			name: "vertex not found",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.ErrorIs(d.AddEdge(mockVertexEID, mockVertexID), ErrVertexNotFound)
				assert.ErrorIs(d.AddEdge(mockVertexID, mockVertexEID), ErrVertexNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID, mockVertexGID, mockVertexHID, mockVertexIID}, nil)
			tc.expect(t, d)
		})
	}
}

func TestDAG_AddEdges(t *testing.T) {
	tests := []struct {
		name   string
		edges  [][2]string
		from   []string
		to     string
		expect func(t *testing.T, d DAG[string], added map[string]struct{})
	}{
		{
			name: "adds edges from several parents at once",
			from: []string{mockVertexEID, mockVertexFID, mockVertexGID},
			to:   mockVertexHID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexEID: {}, mockVertexFID: {}, mockVertexGID: {}}, added)

				vh, err := d.GetVertex(mockVertexHID)
				assert.NoError(err)
				assert.Equal(uint(3), vh.Parents.Len())
				for _, id := range []string{mockVertexEID, mockVertexFID, mockVertexGID} {
					vertex, err := d.GetVertex(id)
					assert.NoError(err)
					assert.True(vh.Parents.Contains(vertex))
					assert.Equal(uint(1), vertex.Children.Len())
				}
			},
		},
		{
			name: "skips self edge",
			from: []string{mockVertexHID, mockVertexEID},
			to:   mockVertexHID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexEID: {}}, added)

				vh, err := d.GetVertex(mockVertexHID)
				assert.NoError(err)
				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(1), vh.Parents.Len())
				assert.True(vh.Parents.Contains(ve))
				assert.Equal(uint(0), vh.Children.Len())
			},
		},
		{
			name: "skips unknown from vertex",
			from: []string{mockVertexID, mockVertexEID},
			to:   mockVertexHID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexEID: {}}, added)

				vh, err := d.GetVertex(mockVertexHID)
				assert.NoError(err)
				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(1), vh.Parents.Len())
				assert.True(vh.Parents.Contains(ve))
			},
		},
		{
			name:  "skips direct and transitive successors that would create a cycle",
			edges: [][2]string{{mockVertexHID, mockVertexEID}, {mockVertexEID, mockVertexFID}},
			from:  []string{mockVertexEID, mockVertexFID, mockVertexGID},
			to:    mockVertexHID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexGID: {}}, added)

				vh, err := d.GetVertex(mockVertexHID)
				assert.NoError(err)
				vg, err := d.GetVertex(mockVertexGID)
				assert.NoError(err)
				assert.Equal(uint(1), vh.Parents.Len())
				assert.True(vh.Parents.Contains(vg))

				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(1), ve.Children.Len())

				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(0), vf.Children.Len())
			},
		},
		{
			name:  "skips an existing edge",
			edges: [][2]string{{mockVertexEID, mockVertexHID}},
			from:  []string{mockVertexEID, mockVertexFID},
			to:    mockVertexHID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexFID: {}}, added)

				vh, err := d.GetVertex(mockVertexHID)
				assert.NoError(err)
				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(2), vh.Parents.Len())
				assert.True(vh.Parents.Contains(ve, vf))
				assert.Equal(uint(1), ve.Children.Len())
			},
		},
		{
			name: "unknown to vertex adds nothing",
			from: []string{mockVertexEID, mockVertexFID},
			to:   mockVertexID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Nil(added)
				for _, id := range []string{mockVertexEID, mockVertexFID} {
					vertex, err := d.GetVertex(id)
					assert.NoError(err)
					assert.Equal(0, vertex.Degree())
				}
			},
		},
		{
			name: "no from vertices adds nothing",
			from: nil,
			to:   mockVertexHID,
			expect: func(t *testing.T, d DAG[string], added map[string]struct{}) {
				assert := assert.New(t)
				assert.Empty(added)

				vh, err := d.GetVertex(mockVertexHID)
				assert.NoError(err)
				assert.Equal(uint(0), vh.Parents.Len())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID, mockVertexGID, mockVertexHID}, tc.edges)
			tc.expect(t, d, d.AddEdges(tc.from, tc.to))
		})
	}
}

func TestDAG_DeleteEdge(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "delete edge",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddEdge(mockVertexEID, mockVertexFID))

				assert.NoError(d.DeleteEdge(mockVertexFID, mockVertexEID))
				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(1), vf.Parents.Len())
				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(1), ve.Children.Len())

				assert.NoError(d.DeleteEdge(mockVertexEID, mockVertexFID))
				vf, err = d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(0), vf.Parents.Len())
				ve, err = d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(0), ve.Children.Len())
			},
		},
		{
			name: "vertex not found",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.ErrorIs(d.DeleteEdge(mockVertexEID, mockVertexID), ErrVertexNotFound)
				assert.ErrorIs(d.DeleteEdge(mockVertexID, mockVertexEID), ErrVertexNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID}, nil)
			tc.expect(t, d)
		})
	}
}

func TestDAG_CanAddEdge(t *testing.T) {
	tests := []struct {
		name   string
		edges  [][2]string
		from   string
		to     string
		expect func(t *testing.T, ok bool)
	}{
		{
			name:  "can add edge",
			edges: [][2]string{{mockVertexEID, mockVertexFID}, {mockVertexFID, mockVertexGID}, {mockVertexFID, mockVertexHID}, {mockVertexGID, mockVertexIID}},
			from:  mockVertexIID,
			to:    mockVertexHID,
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.True(ok)
			},
		},
		{
			name:  "cycle between vertices",
			edges: [][2]string{{mockVertexEID, mockVertexFID}, {mockVertexFID, mockVertexGID}, {mockVertexGID, mockVertexHID}, {mockVertexHID, mockVertexIID}},
			from:  mockVertexIID,
			to:    mockVertexEID,
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "self edge",
			from: mockVertexEID,
			to:   mockVertexEID,
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name:  "edge already exists",
			edges: [][2]string{{mockVertexEID, mockVertexFID}},
			from:  mockVertexEID,
			to:    mockVertexFID,
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "to vertex not found",
			from: mockVertexEID,
			to:   mockVertexID,
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
		{
			name: "from vertex not found",
			from: mockVertexID,
			to:   mockVertexEID,
			expect: func(t *testing.T, ok bool) {
				assert := assert.New(t)
				assert.False(ok)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID, mockVertexGID, mockVertexHID, mockVertexIID}, tc.edges)
			tc.expect(t, d.CanAddEdge(tc.from, tc.to))
		})
	}
}

func TestDAG_CanAddEdges(t *testing.T) {
	tests := []struct {
		name   string
		edges  [][2]string
		from   []string
		to     string
		expect func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int)
	}{
		{
			name: "reports every addable parent",
			from: []string{mockVertexEID, mockVertexFID, mockVertexGID},
			to:   mockVertexHID,
			expect: func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexEID: {}, mockVertexFID: {}, mockVertexGID: {}}, addable)
				assert.Equal(before, after)
			},
		},
		{
			name: "excludes self edge",
			from: []string{mockVertexHID, mockVertexEID},
			to:   mockVertexHID,
			expect: func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexEID: {}}, addable)
				assert.Equal(before, after)
			},
		},
		{
			name: "excludes unknown from vertex",
			from: []string{mockVertexID, mockVertexEID},
			to:   mockVertexHID,
			expect: func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexEID: {}}, addable)
				assert.Equal(before, after)
			},
		},
		{
			name:  "excludes an existing edge",
			edges: [][2]string{{mockVertexEID, mockVertexHID}},
			from:  []string{mockVertexEID, mockVertexFID},
			to:    mockVertexHID,
			expect: func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexFID: {}}, addable)
				assert.Equal(before, after)
			},
		},
		{
			name:  "excludes direct and transitive successors that would create a cycle",
			edges: [][2]string{{mockVertexHID, mockVertexEID}, {mockVertexEID, mockVertexFID}},
			from:  []string{mockVertexEID, mockVertexFID, mockVertexGID},
			to:    mockVertexHID,
			expect: func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int) {
				assert := assert.New(t)
				assert.Equal(map[string]struct{}{mockVertexGID: {}}, addable)
				assert.Equal(before, after)
			},
		},
		{
			name: "unknown to vertex reports nothing",
			from: []string{mockVertexEID, mockVertexFID},
			to:   mockVertexID,
			expect: func(t *testing.T, addable map[string]struct{}, before, after map[string][2]int) {
				assert := assert.New(t)
				assert.Nil(addable)
				assert.Equal(before, after)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID, mockVertexGID, mockVertexHID}, tc.edges)
			before := make(map[string][2]int)
			d.Range(func(id string, vertex *Vertex[string]) bool {
				before[id] = [2]int{vertex.InDegree(), vertex.OutDegree()}
				return true
			})

			addable := d.CanAddEdges(tc.from, tc.to)
			after := make(map[string][2]int)
			d.Range(func(id string, vertex *Vertex[string]) bool {
				after[id] = [2]int{vertex.InDegree(), vertex.OutDegree()}
				return true
			})

			tc.expect(t, addable, before, after)
		})
	}
}

func TestDAG_DeleteVertexInEdges(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "delete inedges",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddEdge(mockVertexEID, mockVertexFID))
				assert.NoError(d.DeleteVertexInEdges(mockVertexFID))

				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(0), vf.Parents.Len())
				assert.Equal(uint(0), vf.Children.Len())
				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(0), ve.Parents.Len())
				assert.Equal(uint(0), ve.Children.Len())
			},
		},
		{
			name: "vertex not found",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.ErrorIs(d.DeleteVertexInEdges(mockVertexID), ErrVertexNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID}, nil)
			tc.expect(t, d)
		})
	}
}

func TestDAG_DeleteVertexOutEdges(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, d DAG[string])
	}{
		{
			name: "delete outedges",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.NoError(d.AddEdge(mockVertexEID, mockVertexFID))
				assert.NoError(d.DeleteVertexOutEdges(mockVertexEID))

				vf, err := d.GetVertex(mockVertexFID)
				assert.NoError(err)
				assert.Equal(uint(0), vf.Parents.Len())
				assert.Equal(uint(0), vf.Children.Len())
				ve, err := d.GetVertex(mockVertexEID)
				assert.NoError(err)
				assert.Equal(uint(0), ve.Parents.Len())
				assert.Equal(uint(0), ve.Children.Len())
			},
		},
		{
			name: "vertex not found",
			expect: func(t *testing.T, d DAG[string]) {
				assert := assert.New(t)
				assert.ErrorIs(d.DeleteVertexOutEdges(mockVertexID), ErrVertexNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, []string{mockVertexEID, mockVertexFID}, nil)
			tc.expect(t, d)
		})
	}
}

func TestDAG_SourceVertices(t *testing.T) {
	tests := []struct {
		name   string
		ids    []string
		edges  [][2]string
		expect func(t *testing.T, vertices []*Vertex[string])
	}{
		{
			name:  "get source vertices",
			ids:   []string{mockVertexEID, mockVertexFID},
			edges: [][2]string{{mockVertexEID, mockVertexFID}},
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(vertices, 1)
				assert.Equal(mockVertexEID, vertices[0].ID)
				assert.Equal(mockVertexValue, vertices[0].Value)
			},
		},
		{
			name: "source vertices not found",
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Empty(vertices)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, tc.ids, tc.edges)
			tc.expect(t, d.GetSourceVertices())
		})
	}
}

func TestDAG_SinkVertices(t *testing.T) {
	tests := []struct {
		name   string
		ids    []string
		edges  [][2]string
		expect func(t *testing.T, vertices []*Vertex[string])
	}{
		{
			name:  "get sink vertices",
			ids:   []string{mockVertexEID, mockVertexFID},
			edges: [][2]string{{mockVertexEID, mockVertexFID}},
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Len(vertices, 1)
				assert.Equal(mockVertexFID, vertices[0].ID)
				assert.Equal(mockVertexValue, vertices[0].Value)
			},
		},
		{
			name: "sink vertices not found",
			expect: func(t *testing.T, vertices []*Vertex[string]) {
				assert := assert.New(t)
				assert.Empty(vertices)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := mockDAG(t, tc.ids, tc.edges)
			tc.expect(t, d.GetSinkVertices())
		})
	}
}

func BenchmarkDAG_AddVertex(b *testing.B) {
	var ids []string
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		ids = append(ids, fmt.Sprint(n))
	}

	b.ResetTimer()
	for _, id := range ids {
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDAG_DeleteVertex(b *testing.B) {
	var ids []string
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}

		ids = append(ids, id)
	}

	b.ResetTimer()
	for _, id := range ids {
		d.DeleteVertex(id)
	}
}

func BenchmarkDAG_GetVertices(b *testing.B) {
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vertices := d.GetVertices()
		if len(vertices) != b.N {
			b.Fatal(errors.New("get vertices failed"))
		}
	}
}

func BenchmarkDAG_GetRandomVertices(b *testing.B) {
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vertices := d.GetRandomVertices(uint(n))
		if len(vertices) != n {
			b.Fatal(errors.New("get random vertices failed"))
		}
	}
}

func BenchmarkDAG_DeleteVertexWithMultiEdges(b *testing.B) {
	var ids []string
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}

		ids = append(ids, id)
	}

	edgeCount := 5
	for index, id := range ids {
		if index+edgeCount > len(ids)-1 {
			break
		}

		for n := 1; n < edgeCount; n++ {
			if err := d.AddEdge(id, ids[index+n]); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for _, id := range ids {
		d.DeleteVertex(id)
	}
}

func BenchmarkDAG_AddEdge(b *testing.B) {
	var ids []string
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}

		ids = append(ids, id)
	}

	b.ResetTimer()
	for index, id := range ids {
		if index < 1 {
			continue
		}

		if err := d.AddEdge(id, ids[index-1]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDAG_AddEdgeWithMultiEdges(b *testing.B) {
	var ids []string
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}

		ids = append(ids, id)
	}

	edgeCount := 5
	for index, id := range ids {
		if index+edgeCount > len(ids)-1 {
			break
		}

		for n := 1; n < edgeCount; n++ {
			if err := d.AddEdge(id, ids[index+n]); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for index, id := range ids {
		if index+edgeCount+1 > len(ids)-1 {
			break
		}

		if err := d.AddEdge(id, ids[index+edgeCount+1]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDAG_DeleteEdge(b *testing.B) {
	var ids []string
	d := NewDAG[string]()
	for n := 0; n < b.N; n++ {
		id := fmt.Sprint(n)
		if err := d.AddVertex(id, string(id)); err != nil {
			b.Fatal(err)
		}

		ids = append(ids, id)
	}

	for index, id := range ids {
		if index < 1 {
			continue
		}

		if err := d.AddEdge(id, ids[index-1]); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for index, id := range ids {
		if index < 1 {
			continue
		}

		if err := d.DeleteEdge(id, ids[index-1]); err != nil {
			b.Fatal(err)
		}
	}
}
