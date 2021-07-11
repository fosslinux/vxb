// SPDX-FileCopyrightText: 2021 fosslinux <fosslinux@aussies.space>
//
// SPDX-License-Identifier: BSD-2-Clause

package vpkgs

import (
    "github.com/fosslinux/vxb/cfg"
    "fmt"
    "errors"
    "os"
    "os/exec"
    str "strings"
    "bufio"
    "sync"
)

// Run an xbps-src command
func XbpsSrc(sArgs string, arch string, mountType string, rtOut bool, cfg cfg.Cfgs) ([]byte, error) {
    var err error
    errRet := make([]byte, 1)
    errRet[0] = 0

    curDir, err := os.Getwd()
    if err != nil {
        return errRet, errors.New("Error getting current working directory")
    }

    err = os.Chdir(cfg.VpkgPath)
    if err != nil {
        return errRet, fmt.Errorf("Unable to change directory into %s", cfg.VpkgPath)
    }

    aArgs := str.Fields(sArgs)

    // Create the masterdir
    // If we are binary-bootstrapping we don't care though
    if aArgs[0] != "binary-bootstrap" {
        // Check a masterdir exists
        _, err := os.Stat(cfg.VpkgPath + "/masterdir/usr")
        if os.IsNotExist(err) {
            return errRet, errors.New("masterdir not bootstrapped")
        }
    }

    // Run the actual command
    var cmd *exec.Cmd
    if cfg.HostArch == arch || aArgs[0] == "binary-bootstrap" {
        // We should not use -a
        cmd = exec.Command("./xbps-src", aArgs...)
    } else {
        cmd = exec.Command("./xbps-src", append([]string{"-a", arch}, aArgs...)...)
    }

    var out []byte
    if rtOut {
        // Get the stdout pipe and stream (bufio) it to our stdout
        stdout, err := cmd.StdoutPipe()
        if err != nil {
            goto errHandler
        }
        stderr, err := cmd.StderrPipe()
        if err != nil {
            goto errHandler
        }

        cmd.Start()

        // Greate goroutines for stdout and stderr so they can be outputted together
        var wg sync.WaitGroup

        wg.Add(2)
        go func(wg *sync.WaitGroup) {
            defer wg.Done()

            scanner := bufio.NewScanner(stdout)
            for scanner.Scan() {
                l := scanner.Text()
                fmt.Println(l)
            }
        }(&wg)

        go func(wg *sync.WaitGroup) {
            defer wg.Done()

            scanner := bufio.NewScanner(stderr)
            for scanner.Scan() {
                l := scanner.Text()
                fmt.Fprintln(os.Stderr, l)
            }
        }(&wg)

        wg.Wait()
        err = cmd.Wait()
        if err != nil {
            goto errHandler
        }

        // We have nothing to return (errRet is just empty)
        out = errRet
    } else {
        out, err = cmd.CombinedOutput()
        if err != nil {
            goto errHandler
        }
    }

    // Cleanup
    os.Chdir(curDir)

    return out, nil

errHandler:
    RemoveMasterdir(cfg)
    fmt.Printf("%s\n", string(out[:]))
    return out, fmt.Errorf("Error %w while executing %s", err, cmd.Args)
}
