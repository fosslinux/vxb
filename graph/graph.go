// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package graph

import (
    "github.com/fosslinux/vxb/cfg"
    "github.com/fosslinux/vxb/vpkgs"
    "github.com/goombaio/dag"
    "fmt"
    "errors"
    str "strings"
)

// Graph struct
type Graph struct {
    g *dag.DAG
    pkgs map[string]*vpkgs.Pkg
}

var pkgGraphError = errors.New("Package already exists in graph")
var pkgRepoError = errors.New("Package is ready in repo")

// Add a package to the graph
func (graphS Graph) addPkg(ident string, cfg cfg.Cfgs) error {
    var err error
    graph := graphS.g

    _, vertexErr := graph.GetVertex(ident)
    // The only type of error returned is a vertex-not-exist error
    if vertexErr == nil {
        return pkgGraphError
    }

    // The vertex does not already exist
    // Run dbulk-dump
    dump, err := vpkgs.DbulkDump(ident, cfg)
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
func (graphS Graph) buildDeps(baseIdent string, cfg cfg.Cfgs) error {
    graph := graphS.g

    var err error

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
    arch := str.Split(baseIdent, "@")[1]
    if cfg.HostArch != arch {
        hostdepends, err = vpkgs.ResolveSubpackages(pkg.Hostmakedepends, cfg.HostArch, cfg)
        if err != nil {
            return err
        }
    }
    if cfg.HostArch != arch {
        depends, err = vpkgs.ResolveSubpackages(append(pkg.Makedepends, pkg.Depends...), arch, cfg)
    } else {
        depends, err = vpkgs.ResolveSubpackages(append(pkg.Hostmakedepends, append(pkg.Makedepends, pkg.Depends...)...), arch, cfg)
    }
    if err != nil {
        return err
    }

    // Start with hostdepends - where we are looking at depName@hostArch
    // Make clear note! We are now in hostArch land! We are *NOT* building
    // ANYTHING for the target arch! This is why arch must be hostArch
    // for all invocations in this block.
    for _, depName := range hostdepends {
        depIdent := depName + "@" + cfg.HostArch
        addPkgErr := graphS.addPkg(depIdent, cfg)
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
            err = graphS.buildDeps(depIdent, cfg)
            if err != nil {
                return err
            }
        }
    }

    // Now, depends (makedepends + depends) - these are handled with
    // depName@arch.
    for _, depName := range depends {
        depIdent := depName + "@" + arch
        addPkgErr := graphS.addPkg(depIdent, cfg)
        if addPkgErr != nil && !errors.Is(addPkgErr, pkgGraphError) && !errors.Is(addPkgErr, pkgRepoError) {
            return err
        }
        // Don't add dep to existing package
        if errors.Is(addPkgErr, pkgRepoError) {
            // Skip to next
            continue
        }
        // Add edge
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
            err = graphS.buildDeps(depIdent, cfg)
            if err != nil {
                return err
            }
        }
    }

    return nil
}

// Generate the graph
func Generate(pkgNames []string, cfg cfg.Cfgs) (Graph, error) {
    var err error

    // Create the DAG + map of pkg dumps
    graph := Graph{g: dag.NewDAG()}
    graph.pkgs = make(map[string]*vpkgs.Pkg)

    // Create the masterdir to be used for all graphing operations
    err = vpkgs.CreateMasterdir(cfg.MountDefault, cfg)
    if err != nil {
        return graph, err
    }

    // Add the initial packages
    for _, pkgName := range pkgNames {
        fmt.Printf("Graphing %s@%s...\n", pkgName, cfg.Arch)
        err = graph.addPkg(pkgName + "@" + cfg.Arch, cfg)
        if errors.Is(err, pkgGraphError) || errors.Is(err, pkgRepoError) {
            // Skip existing packages
            continue
        } else if err != nil {
            // Attempt to remove masterdir
            vpkgs.RemoveMasterdir(cfg)
            return graph, err
        }
        err = graph.buildDeps(pkgName + "@" + cfg.Arch, cfg)
        if err != nil {
            // Attempt to remove masterdir
            vpkgs.RemoveMasterdir(cfg)
            return graph, err
        }
    }

    // Destroy masterdir used
    err = vpkgs.RemoveMasterdir(cfg)
    if err != nil {
        return graph, err
    }

    return graph, nil
}
