/*
 * This is a helper program that mounts a tmpfs/zram/zram-zstd so that we
 * don't need to escalate privileges in the main program.
 */

package main

import (
    "os"
    getoptions "github.com/DavidGamba/go-getoptions"
    "github.com/go-ini/ini"
    "github.com/fosslinux/vxb/util"
    "fmt"
    "strconv"
    "os/user"
)

// Main function
func main() {
    var err error

    // Option parsing
    opt := getoptions.New()
    opt.SetMode(getoptions.Bundling)
    opt.Bool("help", false, opt.Alias("h"))
    var vpkgPath string
    opt.StringVar(&vpkgPath, "vpkg", "", opt.Alias("v"),
        opt.Description("Path to void-packages checkout."))
    var sOwner string
    opt.StringVar(&sOwner, "owner", "", opt.Alias("o"),
        opt.Description("User that the main process will be run as."))
    var unmount bool
    opt.BoolVar(&unmount, "unmount", false, opt.Alias("u"),
        opt.Description("Destroy any current mounts instead of creating them."))
    var confPath string
    opt.StringVar(&confPath, "conf", "conf.ini", opt.Alias("c"),
        opt.Description("Configuration file path."))
    remaining, err := opt.Parse(os.Args[1:])
    if err != nil {
        panic(fmt.Errorf("Error %w while parsing options", err))
    }

    // Process help
    if opt.Called("help") {
        fmt.Fprintf(os.Stderr, opt.Help())
        os.Exit(0)
    }

    // Warn about unhandled arguments
    if len(remaining) != 0 {
        fmt.Fprintf(os.Stderr, "WARN: Unhandled arguments: %v\n", remaining)
    }

    // Check our desired config file exists
    haveConf := true
    _, exists := os.Stat(confPath)
    if os.IsNotExist(exists) {
        if opt.Called("vpkg") {
            fmt.Fprintf(os.Stderr, "ERROR: Config file %s does not exist.", confPath)
            os.Exit(1)
        } else {
            haveConf = false
        }
    }

    // Load the config file
    var iniF *ini.File
    if haveConf {
        iniF, err = ini.Load(confPath)
        if err != nil {
            panic(fmt.Errorf("Error %w encountered while loading config file", err))
        }
    }

    // Load vpkgPath from config file
    if !opt.Called("vpkg") {
        vpkgPath = iniF.Section("vpkg").Key("path").String()
    }
    // We must have a vpkgPath
    if vpkgPath == "" {
        fmt.Fprintf(os.Stderr, "ERROR: No void-packages path provided.\n")
        os.Exit(1)
    }

    // Parse owner
    var owner int
    if sOwner == "" {
        owner = 0
    } else {
        owner, err = strconv.Atoi(sOwner)
        if err != nil {
            // Assume it is a username
            userStruct, err := user.Lookup(sOwner)
            if err != nil {
                panic(fmt.Errorf("Error %w looking up user %s", err, sOwner))
            }
            owner, _ = strconv.Atoi(userStruct.Uid)
        }
    }

    // If any mount type is set in any way within the configuration file
    // then we operate on them.
    mountDefault := iniF.Section("mount").Key("default").String()
    mountPkgs := iniF.Section("mount.pkgs")

    // Starting with tmpfs:
    if mountDefault == "tmpfs" || mountPkgs.HasValue("tmpfs") {
        if unmount {
            err = util.Unmount(vpkgPath + "/mnt/tmpfs")
            if err != nil {
                panic(fmt.Errorf("Error %w occurred unmounting tmpfs on %s", err, vpkgPath + "/mnt/tmpfs"))
            }
        } else {
            os.MkdirAll(vpkgPath + "/mnt/tmpfs", 0755)
            size := iniF.Section("mount").Key("tmpfs_size").String()
            err = util.MountTmpfs(vpkgPath + "/mnt/tmpfs", size)
            if err != nil {
                panic(fmt.Errorf("Error %w occurred mounting tmpfs on %s", err, vpkgPath + "/mnt/tmpfs"))
            }
            err = os.Chown(vpkgPath + "/mnt/tmpfs", owner, -1)
            if err != nil {
                panic(fmt.Errorf("Error %w occurred chowning %s to %s", err, vpkgPath + "/mnt/tmpfs", owner))
            }
        }
    }

    // Now zram:
    if mountDefault == "zram" || mountPkgs.HasValue("zram") {
        if unmount {
            err = util.Unmount(vpkgPath + "/mnt/zram")
            if err != nil {
                panic(fmt.Errorf("Error %w occurred unmounting zram on %s", err, vpkgPath + "/mnt/zram"))
            }
        } else {
            os.MkdirAll(vpkgPath + "/mnt/zram", 0755)
            size := iniF.Section("mount").Key("zram_size").String()
            err = util.MountZram(vpkgPath + "/mnt/zram", size, "lz4")
            if err != nil {
                panic(fmt.Errorf("Error %w occurred mounting zram on %s", err, vpkgPath + "/mnt/zram"))
            }
            err = os.Chown(vpkgPath + "/mnt/zram", owner, -1)
            if err != nil {
                panic(fmt.Errorf("Error %w occurred chowning %s to %s", err, vpkgPath + "/mnt/zram", owner))
            }
        }
    }

    // Now zram-zstd:
    if mountDefault == "zram-zstd" || mountPkgs.HasValue("zram-zstd") {
        if unmount {
            err = util.Unmount(vpkgPath + "/mnt/zram-zstd")
            if err != nil {
                panic(fmt.Errorf("Error %w occurred unmounting zram (zstd) on %s", err, vpkgPath + "/mnt/zram-zstd"))
            }
        } else {
            os.MkdirAll(vpkgPath + "/mnt/zram-zstd", 0755)
            size := iniF.Section("mount").Key("zram_zstd_size").String()
            err = util.MountZram(vpkgPath + "/mnt/zram-zstd", size, "zstd")
            if err != nil {
                panic(fmt.Errorf("Error %w occurred mounting zram (zstd) on %s", err, vpkgPath + "/mnt/zram-zstd"))
            }
            err = os.Chown(vpkgPath + "/mnt/zram-zstd", owner, -1)
            if err != nil {
                panic(fmt.Errorf("Error %w occurred chowning %s to %s", err, vpkgPath + "/mnt/zram-zstd", owner))
            }
        }
    }

    // Chown the base directory
    if !unmount {
        err = os.Chown(vpkgPath + "/mnt", owner, -1)
        if err != nil {
            panic(fmt.Errorf("Error %w occurred chowning %s to %s", err, vpkgPath + "/mnt", owner))
        }
    }

    // Remove the entire folder structure
    if unmount {
        err = os.RemoveAll(vpkgPath + "/mnt")
        if err != nil {
            panic(fmt.Errorf("Error %w occured removing %s", vpkgPath + "/mnt"))
        }
    }
}
