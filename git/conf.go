// SPDX-License-Identifier: BSD-2-Clause

package git

import (
    "errors"
    "fmt"
)

// Git configuration/repo struct
type Repo struct {
    // Path to the git repo
    Path string
    // Git enabled
    Enable bool
    // Git commits to find outdated packages between
    Commits string
    // Branch name
    Branch string
    // Using a remote
    WithRemote bool
    // Remote name
    RemoteName string
    // Remote branch
    RemoteBranch string
    // Strategy to pull in changes from remote
    // Valid: ff, rebase, merge
    RemoteStrategy string
    // Strategy to change commits
    // Valid: rebase, checkout
    CommitStrategy string
    // What do do when merge/rebase/checkout fails
    // Valid: shell, die
    ChangeFail string
}

// Change the commit according to cfg
func (r Repo) changeCommit(commit string) error {
    var err error

    // We have a special value, tip, meaning the tip of the "working" branch,
    // and remote, the tip of the "remote" branch
    if commit == "tip" {
        commit = r.Branch
    } else if commit == "remote" {
        // Check remotes are enabled first
        if !r.WithRemote {
            return errors.New("Can't change to remote that isn't enabled")
        }
        commit = fmt.Sprintf("%s/%s", r.RemoteName, r.RemoteBranch)
    }

    // Perform the action
    switch r.CommitStrategy {
        case "rebase":
            err = r.rebase(commit)
        case "checkout":
            err = r.checkout(commit)
    }

    // Check it worked
    if err != nil {
        switch r.CommitStrategy {
            // We can't do anything about a checkout failure
            case "checkout":
                return err
            // But we can for a rebase
            case "rebase":
                err = r.changeGood(err, true)
                if err != nil {
                    return err
                }
        }
    }

    return nil
}

// Merge/rebase the remote into the branch according to cfg
func (r Repo) reconcileRemote() error {
    var err error

    // Perform the action
    switch r.RemoteStrategy {
        case "rebase":
            err = r.rebase(fmt.Sprintf("%s/%s", r.RemoteName, r.RemoteBranch))
        case "merge":
            err = r.merge()
    }

    // Check it worked
    err = r.changeGood(err, true)
    if err != nil {
        return err
    }

    return nil
}
