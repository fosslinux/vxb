// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
    "github.com/fosslinux/vxb/vpkgs"
    "github.com/fosslinux/vxb/graph"
    getoptions "github.com/DavidGamba/go-getoptions"
    "github.com/go-ini/ini"
    "golang.org/x/sys/unix"
    "errors"
    str "strings"
    "os"
    "fmt"
)

func checkConfValue(val string, errName string) {
    if val == "" {
        // No value was given
        fmt.Fprintf(os.Stderr, "ERROR: %s must be given in config file or cmdline!\n", errName)
        os.Exit(1)
    }
}

func main() {
    var err error

    // Option parsing 
    opt := getoptions.New()
    opt.SetMode(getoptions.Bundling)
    opt.Bool("help", false, opt.Alias("h"))
    var vpkgPath string
    opt.StringVar(&vpkgPath, "vpkg", "", opt.Alias("v"),
        opt.Description("Path to void-packages checkout."))
    var arch string
    opt.StringVar(&arch, "arch", "", opt.Required(), opt.Alias("a"),
        opt.Description("The architecture to build for."))
    var sPkgNames string
    opt.StringVar(&sPkgNames, "pkgname", "", opt.Required(), opt.Alias("p"),
        opt.Description("The package(s) to build."))
    var hostArch string
    opt.StringVar(&hostArch, "hostarch", "", opt.Alias("m"),
        opt.Description("The host architecture."))
    var confFile string
    opt.StringVar(&confFile, "conf", "conf.ini", opt.Alias("c"),
        opt.Description("Configuration file path."))

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
    pkgNames := str.Split(sPkgNames, " ")

    // Config parsing
    // Note this takes a /lower/ priority than option parsing
    // Declare here to bypass stupid goto rule
    var cfg *ini.File
    var utsname unix.Utsname
    _, confFileExists := os.Stat(confFile)
    if os.IsNotExist(confFileExists) {
        if confFile == "conf.ini" {
            // It is the default and hence there is no config file.
            goto finishConfFile
        } else {
            // We were given a bad config file
            fmt.Fprintf(os.Stderr, "ERROR: Cannot open config file %s!\n", confFile)
            os.Exit(1)
        }
    }

    cfg, err = ini.Load(confFile)

    // Path to void-packages
    if vpkgPath == "" {
        vpkgPath = cfg.Section("vpkg").Key("path").String()
        checkConfValue(vpkgPath, "path to void packages")
    }
    // Host architecture
    if hostArch == "" {
        hostArch = cfg.Section("vpkg").Key("host_arch").String()
        // If there is still nothing, use the default logic
        if hostArch == "" {
            err = unix.Uname(&utsname)
            if err != nil {
                panic(errors.New("Error getting uname"))
            }
            hostArch = string(utsname.Machine[:])
        }
    }

    finishConfFile:
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

    fmt.Printf("Generating graph...\n")
    pkgGraph, err := graph.Generate(pkgNames, hostArch, arch, vpkgPath)
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
