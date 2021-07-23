// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
    "github.com/fosslinux/vxb/graph"
    "github.com/fosslinux/vxb/git"
    "github.com/fosslinux/vxb/cfg"
    "os"
    "fmt"
    str "strings"
)

// TODO: A universe build flag will be required for Void that builds all packages.

// Remove an element from a []string
func sStringRm(s []string, i int) []string {
    s[len(s)-1], s[i] = s[i], s[len(s)-1]
    return s[:len(s)-1]
}

// Delete empty lements form a []string
// https://play.golang.org/p/fxVyC4WqjR
func delete_empty(s []string) []string {
	var r []string
	for _, str := range s {
        if str != "" {
            r = append(r, str)
        }
	}
	return r
}

// Generate a package list
func genPkgList(cfg cfg.Cfgs) ([]string, error) {
    var pkgNames []string
    var err error

    // Generate the list of packages
    if cfg.Opt.Called("git") {
        // Generate the updated packages between these commits
        pkgNames, err = git.Changed(cfg.Arch, cfg, str.Split(cfg.Git.Commits, "...")...)
        if err != nil {
            return []string{}, err
        }
        // If we specificed package names as well it is the intersection of
        // those and the updated packages.
        if cfg.Opt.Called("pkgname") {
            validPkgNames := str.Split(cfg.SPkgNames, " ")
            for i, pkgA := range pkgNames {
                found := false
                for _, pkgB := range validPkgNames {
                    if pkgA == pkgB {
                        found = true
                    }
                }
                if !found {
                    pkgNames[i] = ""
                }
            }
            // Now delete all of the empty ones
            pkgNames = delete_empty(pkgNames)
        }
    } else {
        // Then its just package names from the command line
        pkgNames = str.Split(cfg.SPkgNames, " ")
    }

    return pkgNames, nil
}

// Main function
func main() {
    var err error

    // Initalize configuration struct
    cfg := cfg.Cfgs{}

    // Cmdline parsing
    cfg.InitOpt()
    cfg.AddOpts()
    cfg.ActOpts(cfg.Opt.Parse(os.Args[1:]))

    // Config parsing
    // Note this takes a /lower/ priority than option parsing
    hasCfg, err := cfg.InitCfg()
    if err != nil {
        panic(err)
    }
    if hasCfg {
        err = cfg.ParseCfg()
        if err != nil {
            panic(err)
        }
        cfg.ValidVpkgPath()
        cfg.ParseGitCfg()
    } else {
        cfg.ValidVpkgPath()
    }

    // Evaluate bits and pieces
    cfg.EvalAutoMuslExt()

    // Perform validations
    cfg.ValidGitEnabled()

    // Warn if there are modifications NOT being made by default (and we
    // haven't already)
    if !cfg.Opt.Called("mods") && !hasCfg {
        fmt.Fprintf(os.Stderr, "WARN: Assuming there are no local modifications to void-packages.")
    }

    // Do the actual build
    pkgNames, err := genPkgList(cfg)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Generating graph...\n")
    pkgGraph, err := graph.Generate(pkgNames, cfg)
    if err != nil {
        panic(err)
    }
    err = pkgGraph.DagToDot("graph.dot")
    if err != nil {
        panic(err)
    }

    err = pkgGraph.Build(cfg)
    if err != nil {
        panic(err)
    }
}
