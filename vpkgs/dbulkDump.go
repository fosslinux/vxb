// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "github.com/fosslinux/vxb/cfg"
    str "strings"
)

// Pkg struct
type Pkg struct {
    Hostmakedepends []string
    Makedepends     []string
    Depends         []string
    Subpackages     []string
    Ready           bool
}

// Reads a list in the format from dbulk-dump
func readDbulkDumpList(out []string, i int) ([]string, int) {
    var list []string
    for j := 0; str.HasPrefix(out[i], " "); i, j = i + 1, j + 1 {
        list = append(list, str.TrimPrefix(out[i], " "))
    }
    return list, i
}

// Translate dbulk-dump into a readable format
func DbulkDump(ident string, cfg cfg.Cfgs) (Pkg, error) {
    var err error
    var emptyStrSli []string = nil

    // Create the pkg to be returned
    pkg := Pkg{}

    // Check if the package is ready
    pkg.Ready, err = pkgReady(ident, cfg)
    if err != nil {
        return pkg, err
    }

    // Execute dbulk-dump
    pkgName := str.Split(ident, "@")[0]
    arch := str.Split(ident, "@")[1]
    bOut, err := XbpsSrc("dbulk-dump " + pkgName, arch, cfg.MountDefault, false, cfg)
    if err != nil {
        return Pkg{}, err
    }
    out := str.Split(string(bOut[:]), "\n")

    // Parse dbulk-dump
    i := 3
    // Skip pkgName, version, revision
    // Skip bootstrap (if it exists!)
    if str.HasPrefix(out[i], "bootstrap: ") {
        i++
    }
    // Read hostmakedepends (if it exists)
    if out[i] == "hostmakedepends:" {
        i++
        pkg.Hostmakedepends, i = readDbulkDumpList(out, i)
    } else {
        pkg.Hostmakedepends = emptyStrSli
    }
    // Read makedepends (if it exists)
    if out[i] == "makedepends:" {
        i++
        pkg.Makedepends, i = readDbulkDumpList(out, i)
    } else {
        pkg.Makedepends = emptyStrSli
    }
    // Read depends (if it exists)
    if out[i] == "depends:" {
        i++
        pkg.Depends, i = readDbulkDumpList(out, i)
    } else {
        pkg.Depends = emptyStrSli
    }
    // Read subpackages (if it exists)
    if out[i] == "subpackages:" {
        i++
        pkg.Subpackages, i = readDbulkDumpList(out, i)
    } else {
        pkg.Subpackages = emptyStrSli
    }

    return pkg, nil
}
