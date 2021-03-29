// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "fmt"
    "os"
)

// Check if a package is a subpackage
func isSubpackage(pkgName string, hostArch string, arch string, vpkgPath string) (bool, error) {
    // dbulk-dump the supposed subpackage
    dump, err := DbulkDump(pkgName, hostArch, arch, vpkgPath)
    if err != nil {
        return false, fmt.Errorf("%w finding if %s is a subpackage", err, pkgName)
    }

    // If the pkgname is in the subpackages, we know it is a subpackage
    for _, line := range dump.Subpackages {
        if line == pkgName {
            return true, nil
        }
    }
    return false, nil
}

// Resolve a subpackage to its base package
func ResolveSubpackage(pkgName string, hostArch string, arch string, vpkgPath string) (string, error) {
    var err error
    // If it isn't a subpackage we don't need to resolve anything
    pkgIsSubpkg, err := isSubpackage(pkgName, hostArch, arch, vpkgPath)
    if err != nil {
        return "", err
    }
    if !pkgIsSubpkg {
        return pkgName, nil
    }

    // Ok, so it is a subpackage
    // Read the link in srcpkgs/ to determine the base package
    rslvPath := vpkgPath + "/srcpkgs/" + pkgName
    basePkg, err := os.Readlink(rslvPath)
    if err != nil {
        return "", fmt.Errorf("Error %w resolving %s", err, rslvPath)
    }
    return basePkg, nil
}

// Resolve subpackages in array to base packages
func ResolveSubpackages(pkgNames []string, hostArch string, arch string, vpkgPath string) ([]string, error) {
    var basePkgs []string
    // Loop over array
    for _, pkgName := range pkgNames {
        basePkg, err := ResolveSubpackage(pkgName, hostArch, arch, vpkgPath)
        if err != nil {
            return []string{}, err
        }
        // Make sure it is not a duplicate
        isDuplicate := false
        for _, otherPkg := range basePkgs {
            if otherPkg == basePkg {
                // Skip
                isDuplicate = true
            }
        }
        if isDuplicate {
            continue
        }
        basePkgs = append(basePkgs, basePkg)
    }
    return basePkgs, nil
}
