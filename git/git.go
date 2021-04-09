// SPDX-License-Identifier: BSD-2-Clause

package git

import (
    "fmt"
    "os"
    "os/exec"
    str "strings"
    "errors"
)

// Execute a git command
func (r Repo) git(sArgs string) ([]byte, error) {
    var err error
    // TODO: functionality below is very similar to xbpsSrc.go
    // Consolidate into one global function?
    errRet := []byte{}

    curDir, err := os.Getwd()
    if err != nil {
        return errRet, errors.New("Error getting current working dir")
    }

    err = os.Chdir(r.Path)
    if err != nil {
        return errRet, fmt.Errorf("Unable to change directory into %s", r.Path)
    }

    // Run the actual command
    cmd := exec.Command("git", str.Fields(sArgs)...)
    out, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Printf("%s\n", string(out[:]))
        return out, fmt.Errorf("Error %w while executing %s", err, cmd.Args)
    }

    err = os.Chdir(curDir)
    if err != nil {
        return errRet, fmt.Errorf("Unable to change directory into %s", curDir)
    }

    return out, nil
}

// Fetch upstream
func (r Repo) fetch() error {
    _, err := r.git(fmt.Sprintf("fetch %s %s", r.RemoteName, r.RemoteBranch))
    if err != nil {
        return err
    }
    return nil
}

// Checkout to a commit
func (r Repo) checkout(commit string) error {
    // Checkout
    _, err := r.git("checkout " + commit)
    if err != nil {
        return err
    }
    return nil
}

// Rebase "work" branch on a specific commit
func (r Repo) rebase(commit string) error {
    var err error

    // We first need to be on the tip of the branch we are rebasing in 
    err = r.checkout(r.Branch)
    if err != nil {
        return err
    }

    _, err = r.git(fmt.Sprintf("rebase --onto %s", commit))
    if err != nil {
        return err
    }
    return nil
}

// Merge the remote into the branch
func (r Repo) merge() error {
    var err error

    // We first need to be on the tip of the branch we are merging into
    err = r.checkout(r.Branch)
    if err != nil {
        return err
    }

    _, err = r.git(fmt.Sprintf("merge %s/%s", r.RemoteName, r.RemoteBranch))
    if err != nil {
        return err
    }

    return nil
}

// Check if a rebase is in progress
// Assumes already chdir()d into the directory
func (r Repo) rebaseInProgress() bool {
    var err error

    _, err = os.Stat(".git/rebase-merge")
    if !os.IsNotExist(err) {
        return true
    }
    _, err = os.Stat(".git/rebase-apply")
    if !os.IsNotExist(err) {
        return true
    }

    return false
}

// Check if merge is in progress
// Assumes already chdir()d into the directory
func (r Repo) mergeInProgress() bool {
    _, err := os.Stat(".git/MERGE_HEAD")
    if !os.IsNotExist(err) {
        return true
    }
    return false
}

// Check if a rebase/merge is "good" - if not, give the user a change to fix it
func (r Repo) changeGood(rErr error, initRun bool) error {
    var err error

    // If there is no error, then we are fine
    if rErr == nil {
        return nil
    }

    // Verb of the change for errors
    var doing string
    switch r.CommitStrategy {
        case "rebase":
            doing = "rebasing"
        case "merge":
            doing = "merging"
    }

    // Are we configured to drop to a shell, or to just exit?
    if r.ChangeFail == "die" {
        return fmt.Errorf("Error %w while %s!\n", rErr, doing)
    }

    // Ok, so we need to drop to a shell
    if initRun {
        fmt.Fprintf(os.Stderr, "ERROR: %s while %s!\n", rErr, doing)
    }

    fmt.Printf("Dropping to a shell to allow you to fix.\n")
    fmt.Printf("To exit vxb, exit with a non-zero return code.\n")
    curDir, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("Unable to get current directory with %w", err)
    }
    err = os.Chdir(r.Path)
    if err != nil {
        return fmt.Errorf("Unable to change directory into %s with %w", r.Path, err)
    }
    cmd := exec.Command("sh")
    cmd.Stdout = os.Stdout
    cmd.Stdin = os.Stdin
    cmd.Stderr = os.Stderr
    err = cmd.Run()

    // If we errored out, then exit
    if err != nil {
        err = os.Chdir(curDir)
        if err != nil {
            return fmt.Errorf("Unable to change directory into %s with %w", curDir, err)
        }
        return fmt.Errorf("%s fix failed :(", r.CommitStrategy)
    }

    // But if we didn't actually finish fixing it, do all this again
    if r.rebaseInProgress() || r.mergeInProgress() {
        fmt.Printf("You did not successfully fix the %s!\n", r.CommitStrategy)
        fmt.Printf("Lets try again...\n")
        err = r.changeGood(rErr, false)
        if err != nil {
            return err
        }
    }

    return nil
}
