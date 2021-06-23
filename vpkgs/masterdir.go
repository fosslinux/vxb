// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "github.com/fosslinux/vxb/util"
    "os"
    "fmt"
)

// Create (i.e. binary-bootstrap) a masterdir
func CreateMasterdir(vpkgPath string, arch string, mountType string, size string) error {
    var err error

    // Check if we need to handle mounting the masterdir
    switch mountType {
    case "none":
        // No processing is required
        break
    case "tmpfs":
        err = util.MountTmpfs("masterdir", size)
        if err != nil {
            return err
        }
    case "zram":
        // Default is lz4
        err = util.MountZram("masterdir", size, "lz4")
        if err != nil {
            return err
        }
    case "zram-zstd":
        err = util.MountZram("masterdir", size, "zstd")
        if err != nil {
            return err
        }
    }

    // Bootstrap the actual masterdir
    _, err = XbpsSrc(vpkgPath, arch, arch, "binary-bootstrap " + arch, false)
    if err != nil {
        return err
    }
    return nil
}

// Remove a masterdir
func RemoveMasterdir(vpkgPath string) error {
    var err error

    // Unmount it if it is mounted
    util.Unmount(vpkgPath + "/masterdir")

    // Remove remainders
    err = os.RemoveAll(vpkgPath + "/masterdir")
    if err != nil {
        return fmt.Errorf("Unable to remove masterdir with %w", err)
    }

    return nil
}
