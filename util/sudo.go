package util

import (
    "os/exec"
    "errors"
)

// Create a command to be run as root
func Sudo(cmd string, args ...string) (*exec.Cmd, error) {
    var err error
    errRet := exec.Command("")

    // Check for sudo and doas
    var ourSudo string
    if _, err = exec.LookPath("sudo"); err == nil {
        ourSudo = "sudo"
    } else if _, err = exec.LookPath("doas"); err == nil {
        ourSudo = "doas"
    } else {
        return errRet, errors.New("Unable to find privildge escalation tool!")
    }

    return exec.Command(ourSudo, append([]string{cmd}, args...)...), nil
}
