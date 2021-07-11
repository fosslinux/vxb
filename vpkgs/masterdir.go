// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "github.com/fosslinux/vxb/cfg"
    "os"
    "fmt"
)

// Create (i.e. binary-bootstrap) a masterdir
func CreateMasterdir(mountType string, cfg cfg.Cfgs) error {
    var err error

    // Check if we need to handle different types of masterdirs
    if mountType == "none" {
        // Make the actual directory
        err = os.Mkdir(cfg.VpkgPath + "/masterdir", 0755)
        if err != nil {
            return fmt.Errorf("Unable to create masterdir directory with %w", err)
        }
    } else {
        // Make the appropriate symlink
        err = os.Symlink("mnt/" + mountType, cfg.VpkgPath + "/masterdir")
        if err != nil {
            return fmt.Errorf("Unable to create symlink for masterdir with %w", err)
        }
    }

    // Bootstrap the actual masterdir
    _, err = XbpsSrc("binary-bootstrap " + cfg.HostArch, cfg.HostArch, "", false, cfg)
    if err != nil {
        return err
    }
    return nil
}

// Remove a masterdir
func RemoveMasterdir(cfg cfg.Cfgs) error {
    var err error

    // Remove all subdirectories/files
    vpkgDir, err := os.Open(cfg.VpkgPath + "/masterdir")
    if err != nil {
        return fmt.Errorf("Error %w opening %s", err, cfg.VpkgPath + "/masterdir")
    }
    within, err := vpkgDir.Readdir(0)
    if err != nil {
        return fmt.Errorf("Error %w listing subfiles/directories within %s", err, cfg.VpkgPath + "/masterdir")
    }
    for _, f := range within {
        // Remove each
        err = os.RemoveAll(cfg.VpkgPath + "/masterdir/" + f.Name())
        if err != nil {
            return fmt.Errorf("Unable to remove %s with %w", cfg.VpkgPath + "/masterdir/" + f.Name(), err)
        }
    }

    // Finally, remove the directory itself
    err = os.RemoveAll(cfg.VpkgPath + "/masterdir")
    if err != nil {
        return fmt.Errorf("Unable to remove masterdir with %w", err)
    }

    return nil
}
