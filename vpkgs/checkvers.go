// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "os"
    "os/exec"
    str "strings"
    "fmt"
)

// Vers struct
type Vers struct {
    all []string
    outdated []string
}

// Run xbps-checkvers for all packages
func checkversAll(baseArgs []string) ([]string, error) {
    args := append(baseArgs, "-s")
    cmd := exec.Command("xbps-checkvers", args...)
    out, err := cmd.Output()
    if err != nil {
        return []string{}, fmt.Errorf("Error %w while running %v", err, cmd.Args)
    }
    return str.Split(string(out[:]), "\n"), nil
}

// Run xbps-checkvers for outdated pacakges
func checkversOutdated(baseArgs []string) ([]string, error) {
    cmd := exec.Command("xbps-checkvers", baseArgs...)
    out, err := cmd.Output()
    if err != nil {
        return []string{}, fmt.Errorf("Error %w while running %v", err, cmd.Args)
    }
    return str.Split(string(out[:]), "\n"), nil
}

// Create a Vers struct
func checkvers(arch string, vpkgPath string) (Vers, error) {
    var err error
    vers := Vers{}
    baseArgs := []string{"-D", vpkgPath, "-R", vpkgPath + "/hostdir/binpkgs", "-i"}

    // Set XBPS_TARGET_ARCH
    err = os.Setenv("XBPS_TARGET_ARCH", arch)
    if err != nil {
        return vers, fmt.Errorf("Error %w setting XBPS_TARGET_ARCH=%s", err, arch)
    }

    // Get xbps-checkvers for all pkgs
    vers.all, err = checkversAll(baseArgs)
    if err != nil {
        return vers, err
    }

    // Get xbps-checkvers for outdated pkgs
    vers.outdated, err = checkversOutdated(baseArgs)
    if err != nil {
        return vers, err
    }

    return vers, nil
}

// Get the state of a package - either ready (true) or not (false)
func pkgReady(pkgName string, arch string, vpkgPath string) (bool, error) {
    var err error
    vers, err := checkvers(arch, vpkgPath)
    if err != nil {
        return false, err
    }

    pkgName = pkgName + " "

    // First, handle case of present and not up-to-date (updated package)
    for _, line := range vers.outdated {
        if str.HasPrefix(line, pkgName) {
            return false, nil
        }
    }

    // Next, not present (new package)
    for _, line := range vers.all {
        if str.HasPrefix(line, pkgName + "?") {
            return false, nil
        }
    }

    // If we get here, it must be present and up-to-date
    return true, nil
}
