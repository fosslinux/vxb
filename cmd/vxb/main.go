// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
    "github.com/fosslinux/vxb/vpkgs"
    "github.com/fosslinux/vxb/graph"
    getoptions "github.com/DavidGamba/go-getoptions"
    "golang.org/x/sys/unix"
    "errors"
    str "strings"
    "os"
    "fmt"
)

func main() {
    var err error

    // Option parsing 
    opt := getoptions.New()
    opt.SetMode(getoptions.Bundling)
    opt.Bool("help", false, opt.Alias("h"))
    var vpkgPath string
    opt.StringVar(&vpkgPath, "vpkg", "", opt.Required(), opt.Alias("v"),
        opt.Description("Path to void-packages checkout."))
    var arch string
    opt.StringVar(&arch, "arch", "", opt.Required(), opt.Alias("a"),
        opt.Description("The architecture to build for."))
    var pkgName string
    opt.StringVar(&pkgName, "pkgname", "", opt.Required(), opt.Alias("p"),
        opt.Description("The package to build."))
    var hostArch string
    // Get the host architecture
    var utsname unix.Utsname
    err = unix.Uname(&utsname)
    if err != nil {
        panic(errors.New("Error getting uname"))
    }
    opt.StringVar(&hostArch, "hostarch", string(utsname.Machine[:]),
        opt.Alias("m"), opt.Description("The host architecture."))

    // Go parse!
    remaining, err := opt.Parse(os.Args[1:])
    if opt.Called("help") {
        fmt.Fprintf(os.Stderr, opt.Help())
        os.Exit(0)
    }
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
        fmt.Fprintf(os.Stderr, opt.Help(getoptions.HelpSynopsis))
        os.Exit(1)
    }
    if len(remaining) != 0 {
        fmt.Fprintf(os.Stderr, "Unhandled arguments: %v\n", remaining)
    }

    // If the architecture is -musl and the host was not manually set, then
    // the host should also be -musl.
    if str.HasSuffix(arch, "-musl") && hostArch != string(utsname.Machine[:]) {
        hostArch += "-musl"
    }

    // Validate options
    validArchs := []string{"aarch64", "armv5tel", "armv6l", "armv7l", "i686",
        "mips-musl", "mipsel-musl", "mipselhf-musl", "mipshf-musl", "ppc",
        "ppc64", "ppc64le", "ppcle", "x86_64"}
    for _, arch := range validArchs {
        if !str.HasSuffix(arch, "-musl") {
            validArchs = append(validArchs, arch + "-musl")
        }
    }
    hostFound := false
    for _, tArch := range validArchs {
        if tArch == hostArch {
            hostFound = true
            break
        }
    }
    if !hostFound {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid architecture.", hostArch)
        os.Exit(1)
    }
    archFound := false
    for _, tArch := range validArchs {
        if tArch == arch {
            archFound = true
            break
        }
    }
    if !archFound {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid architecture.", arch)
        os.Exit(1)
    }

    // Do the actual build
    fmt.Printf("Creating masterdir in %s for %s...\n", vpkgPath + "/masterdir", hostArch)
    err = vpkgs.RemoveMasterdir(vpkgPath)
    if err != nil {
        panic(err)
    }
    err = vpkgs.CreateMasterdir(vpkgPath, hostArch)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Generating graph for %s@%s...\n", pkgName, arch)
    pkgGraph, err := graph.Generate(pkgName, hostArch, arch, vpkgPath)
    if err != nil {
        panic(err)
    }
    err = pkgGraph.DagToDot("graph.dot")
    if err != nil {
        panic(err)
    }

    err = pkgGraph.Build(hostArch, vpkgPath)
    if err != nil {
        panic(err)
    }
}
