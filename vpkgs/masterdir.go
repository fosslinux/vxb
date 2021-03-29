// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "os"
    "fmt"
)

// Create (i.e. binary-bootstrap) a masterdir
func CreateMasterdir(vpkgPath string, arch string) error {
    _, err := XbpsSrc(vpkgPath, arch, arch, "binary-bootstrap " + arch)
    if err != nil {
        return err
    }
    return nil
}

// Remove a masterdir
func RemoveMasterdir(vpkgPath string) error {
    err := os.RemoveAll(vpkgPath + "masterdir")
    if err != nil {
        return fmt.Errorf("Unable to remove masterdir with %w", err)
    }
    return nil
}
