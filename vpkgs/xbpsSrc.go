// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "fmt"
    "errors"
    "os"
    "os/exec"
    str "strings"
)

// Run an xbps-src command
func XbpsSrc(vpkgPath string, hostArch string, arch string, sArgs string) ([]byte, error) {
    var err error
    errRet := make([]byte, 1)
    errRet[0] = 0

    curDir, err := os.Getwd()
    if err != nil {
        return errRet, errors.New("Error getting current working directory")
    }

    err = os.Chdir(vpkgPath)
    if err != nil {
        return errRet, fmt.Errorf("Unable to change directory into %s", vpkgPath)
    }

    aArgs := str.Fields(sArgs)

    // Create the masterdir if it dosen't exist
    // If we are binary-bootstrapping we don't care though
    if aArgs[0] != "binary-bootstrap" {
        _, checkA := os.Stat("masterdir")
        // An arbitary path that exists inside a working masterdir
        _, checkB := os.Stat("masterdir/usr")
        if os.IsNotExist(checkA) || os.IsNotExist(checkB) {
            // The masterdir dosen't exist
            // Kill off anything that already exists
            err = RemoveMasterdir(vpkgPath)
            if err != nil {
                return errRet, err
            }
            // Actually make the masterdir
            err = CreateMasterdir(vpkgPath, hostArch)
            if err != nil {
                return errRet, err
            }
        }
    }

    // Run the actual command
    var cmd *exec.Cmd
    if hostArch == arch || aArgs[0] == "binary-bootstrap" {
        // We should not use -a
        cmd = exec.Command("./xbps-src", aArgs...)
    } else {
        cmd = exec.Command("./xbps-src", append([]string{"-a", arch}, aArgs...)...)
    }
    out, err := cmd.CombinedOutput()
    if err != nil {
        os.RemoveAll("masterdir")
        fmt.Printf("%s\n", string(out[:]))
        return out, fmt.Errorf("Error %w while executing %s %v", err, cmd.Args)
    }

    // Cleanup
    os.Chdir(curDir)

    return out, nil
}
