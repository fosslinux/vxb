// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package graph

import (
    "github.com/fosslinux/vxb/vpkgs"
    "github.com/goombaio/dag"
    "fmt"
    "errors"
)

// Graph struct
type Graph struct {
    g *dag.DAG
    pkgs map[string]*vpkgs.Pkg
}

var pkgGraphError = errors.New("Package already exists in graph")
var pkgRepoError = errors.New("Package is ready in repo")

// Add a package to the graph
func (graphS Graph) addPkg(pkgName string, hostArch string, arch string, vpkgPath string) error {
    var err error
    graph := graphS.g

    ident := pkgName + "@" + arch

    _, vertexErr := graph.GetVertex(ident)
    // The only type of error returned is a vertex-not-exist error
    if vertexErr == nil {
        return pkgGraphError
    }

    // The vertex does not already exist
    // Run dbulk-dump
    dump, err := vpkgs.DbulkDump(pkgName, hostArch, arch, vpkgPath)
    if err != nil {
        return fmt.Errorf("%w adding ident to graph", err, ident)
    }
    // Add the dump to pkgs map under identifier
    graphS.pkgs[ident] = &dump

    // Check if package is already ready
    if graphS.pkgs[ident].Ready {
        return pkgRepoError
    }

    // Add to graph - make the identifier pkgname@arch
    err = graph.AddVertex(dag.NewVertex(ident, nil))
    if err != nil {
        return fmt.Errorf("Error %w adding vertex %s", err, ident)
    }

    return nil
}

// Build dependencies of a package into the graph (recursively)
func (graphS Graph) buildDeps(pkgName string, hostArch string, arch string, vpkgPath string) error {
    graph := graphS.g

    var err error
    baseIdent := pkgName + "@" + arch

    baseVertex, err := graph.GetVertex(baseIdent)
    if err != nil {
        return fmt.Errorf("Error %w getting vertex %s", err, baseIdent)
    }
    pkg := graphS.pkgs[baseIdent]

    // Resolve all subpackages + concat arays appropriatley
    // If we are cross-building (host != target) then we need hostdepends
    // and depends. If we are native building (host == target) then we
    // only should have depends to avoid duplicates.
    var hostdepends []string
    var depends []string
    if hostArch != arch {
        hostdepends, err = vpkgs.ResolveSubpackages(pkg.Hostmakedepends, hostArch, hostArch, vpkgPath)
        if err != nil {
            return err
        }
    }
    if hostArch != arch {
        depends, err = vpkgs.ResolveSubpackages(append(pkg.Makedepends, pkg.Depends...), hostArch, arch, vpkgPath)
    } else {
        depends, err = vpkgs.ResolveSubpackages(append(pkg.Hostmakedepends, append(pkg.Makedepends, pkg.Depends...)...), hostArch, arch, vpkgPath)
    }
    if err != nil {
        return err
    }

    // Start with hostdepends - where we are looking at depName@hostArch
    // Make clear note! We are now in hostArch land! We are *NOT* building
    // ANYTHING for the target arch! This is why arch must be hostArch
    // for all invocations in this block.
    for _, depName := range hostdepends {
        addPkgErr := graphS.addPkg(depName, hostArch, hostArch, vpkgPath)
        if addPkgErr != nil && !errors.Is(addPkgErr, pkgGraphError) && !errors.Is(addPkgErr, pkgRepoError) {
            return err
        }
        // If the package is already in the repo, we DON'T want to add a
        // dep to it.
        if errors.Is(addPkgErr, pkgRepoError) {
            // Skip to next package
            continue
        }
        // Add the edge
        depIdent := depName + "@" + hostArch
        depVertex, err := graph.GetVertex(depIdent)
        if err != nil {
            return fmt.Errorf("Error %w fetching vertex %s", err, depIdent)
        }
        err = graph.AddEdge(baseVertex, depVertex)
        if err != nil {
            return fmt.Errorf("Error %w adding edge for %s -> %s", err, baseIdent, depIdent)
        }
        // Recursively build dependencies
        // Don't build it's deps if it already exists in the graph - no need
        // to repeat that work.
        if !errors.Is(addPkgErr, pkgGraphError) {
            err = graphS.buildDeps(depName, hostArch, hostArch, vpkgPath)
            if err != nil {
                return err
            }
        }
    }

    // Now, depends (makedepends + depends) - these are handled with
    // depName@arch.
    for _, depName := range depends {
        addPkgErr := graphS.addPkg(depName, hostArch, arch, vpkgPath)
        if addPkgErr != nil && !errors.Is(addPkgErr, pkgGraphError) && !errors.Is(addPkgErr, pkgRepoError) {
            return err
        }
        // Don't add dep to existing package
        if errors.Is(addPkgErr, pkgRepoError) {
            // Skip to next
            continue
        }
        // Add edge
        depIdent := depName + "@" + arch
        depVertex, err := graph.GetVertex(depIdent)
        if err != nil {
            return fmt.Errorf("Error %w fetching vertex %s", err, depIdent)
        }
        err = graph.AddEdge(baseVertex, depVertex)
        if err != nil {
            return fmt.Errorf("Error %w adding edge for %s -> %s", err, baseIdent, depIdent)
        }
        // Recurse
        if !errors.Is(addPkgErr, pkgGraphError) {
            err = graphS.buildDeps(depName, hostArch, arch, vpkgPath)
            if err != nil {
                return err
            }
        }
    }

    return nil
}

// Generate the graph
func Generate(pkgNames []string, hostArch string, arch string, vpkgPath string) (Graph, error) {
    var err error

    // Create the DAG + map of pkg dumps
    graph := Graph{g: dag.NewDAG()}
    graph.pkgs = make(map[string]*vpkgs.Pkg)

    // Add the initial packages
    for _, pkgName := range pkgNames {
        fmt.Printf("Graphing %s@%s...\n", pkgName, arch)
        err = graph.addPkg(pkgName, hostArch, arch, vpkgPath)
        if errors.Is(err, pkgGraphError) || errors.Is(err, pkgRepoError) {
            // Skip existing packages
            continue
        } else if err != nil {
            return graph, err
        }
        err = graph.buildDeps(pkgName, hostArch, arch, vpkgPath)
        if err != nil {
            return graph, err
        }
    }

    return graph, nil
}
