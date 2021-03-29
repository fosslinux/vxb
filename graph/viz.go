// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package graph

import (
    "github.com/goombaio/dag"
    "fmt"
    "os"
)

// https://play.golang.org/p/Qg_uv_inCek
// Check if a []*dag.Vertex slice contains a value
func sliceContains(s []*dag.Vertex, v *dag.Vertex) bool {
	for _, i := range s {
		if i == v {
			return true
		}
	}
	return false
}

// Depth-first traversal from vertex
func (graphS Graph) traverse(f *os.File, vertex *dag.Vertex, done *[]*dag.Vertex, fname string) error {
    graph := graphS.g

    var err error
    // Check if we are in done[]; if we are, we don't need to do anything
    if sliceContains(*done, vertex) {
        return nil
    }

    // We set this here to avoid loops
    *done = append(*done, vertex)

    // Loop over children
    children, err := graph.Successors(vertex)
    if err != nil {
        return fmt.Errorf("Unable to get children of %s with %w", vertex.ID, err)
    }

    for _, child := range children {
        // Add the line to the DOT
        _, err = f.WriteString(fmt.Sprintf("\"%s\" -> \"%s\"\n", vertex.ID, child.ID))
        if err != nil {
            return fmt.Errorf("Unable to write to %s with %w", fname, err)
        }
        // Recurse to children
        err = graphS.traverse(f, child, done, fname)
        if err != nil {
            return err
        }
    }

    return nil
}

// Write a DAG graph to a DOT file
func (graphS Graph) DagToDot(fname string) error {
    graph := graphS.g

    var err error
    // An array for all the vertexes which we have written
    var done []*dag.Vertex

    f, err := os.Create(fname)
    if err != nil {
        return fmt.Errorf("Unable to open %s for writing with %w", fname, err)
    }
    defer f.Close()

    // Add header
    _, err = f.WriteString("digraph {\n")
    if err != nil {
        return fmt.Errorf("Unable to write to %s with %w", fname, err)
    }

    // Loop over source verticies
    for _, vertex := range graph.SourceVertices() {
        _, err = f.WriteString(fmt.Sprintf("base -> \"%s\"\n", vertex.ID))
        if err != nil {
            return fmt.Errorf("Unable to write to %s with %w", fname, err)
        }
        // Go to children
        err = graphS.traverse(f, vertex, &done, fname)
        if err != nil {
            return err
        }
    }

    // Add footer
    _, err = f.WriteString("}\n")
    if err != nil {
        return fmt.Errorf("Unable to write to %s with %w", fname, err)
    }

    return nil
}
