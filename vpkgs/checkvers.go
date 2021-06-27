// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "github.com/fosslinux/vxb/cfg"
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
func checkvers(arch string, cfg cfg.Cfgs) (Vers, error) {
    var err error
    vers := Vers{}
    baseArgs := []string{"-D", cfg.VpkgPath, "-R", cfg.VpkgPath + "/hostdir/binpkgs", "-i"}

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
    // Remove the dumb empty element on the end
    vers.outdated = vers.outdated[:len(vers.outdated) - 1]
    if err != nil {
        return vers, err
    }

    return vers, nil
}

// Get the state of a package - either ready (true) or not (false)
func pkgReady(ident string, cfg cfg.Cfgs) (bool, error) {
    var err error
    arch := str.Split(ident, "@")[1]
    vers, err := checkvers(arch, cfg)
    if err != nil {
        return false, err
    }

    pkgName := str.Split(ident, "@")[0] + ""

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

// Check if there are any not up to date packages, and list them
func Ready(arch string, cfg cfg.Cfgs) (bool, []string, error) {
    // Run checkvers
    vers, err := checkvers(arch, cfg)
    if err != nil {
        return false, []string{}, err
    }

    // Make the list
    outdated := make([]string, len(vers.outdated))
    for i, line := range(vers.outdated) {
        outdated[i] = str.Split(line, " ")[0]
    }

    return len(outdated) == 0, outdated, nil
}
