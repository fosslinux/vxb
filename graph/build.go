// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package graph

import (
    "github.com/goombaio/dag"
    "github.com/fosslinux/vxb/vpkgs"
    "fmt"
    str "strings"
)

// Build a particular package
func build(ident string, hostArch string, vpkgPath string) error {
    var err error

    splitIdent := str.Split(ident, "@")
    pkgname := splitIdent[0]
    arch := splitIdent[1]

    // Go!
    args := "pkg -N " + pkgname
    out, err := vpkgs.XbpsSrc(vpkgPath, hostArch, arch, args)
    fmt.Printf("%v", string(out))
    if err != nil {
        return fmt.Errorf("%w building %s", err, ident)
    }

    return nil
}

// Build the children of a vertex
func (graphS Graph) children(vertex *dag.Vertex, hostArch string, vpkgPath string) error {
    graph := graphS.g
    pkgs := graphS.pkgs

    var err error

    // Loop over children
    children, err := graph.Successors(vertex)
    if err != nil {
        return fmt.Errorf("Unable to get children of %s with %w", vertex.ID, err)
    }
    for _, child := range children {
        // Check if we are ready
        if pkgs[child.ID].Ready {
            // We don't need to do anything
            continue
        }

        // Children first again!
        err = graphS.children(child, hostArch, vpkgPath)
        if err != nil {
            return err
        }

        // Build this package
        fmt.Printf("Building %s (pulled in by %s)...\n", child.ID, vertex.ID)
        err = build(child.ID, hostArch, vpkgPath)
        if err != nil {
            return err
        }

        pkgs[child.ID].Ready = true
    }

    return nil
}

// Build packages in graph
func (graphS Graph) Build(hostArch string, vpkgPath string) error {
    graph := graphS.g
    var err error

    // Loop over source vertices (which are never ready)
    for _, vertex := range graph.SourceVertices() {
        // Children first
        err = graphS.children(vertex, hostArch, vpkgPath)
        if err != nil {
            return err
        }

        // Now we can build
        fmt.Printf("Building %s...\n", vertex.ID)
        err = build(vertex.ID, hostArch, vpkgPath)
        if err != nil {
            return err
        }
        graphS.pkgs[vertex.ID].Ready = true
    }

    return nil
}
