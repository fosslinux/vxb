// SPDX-License-Identifier: BSD-2-Clause

package git

import (
    "github.com/fosslinux/vxb/cfg"
    "errors"
    "fmt"
)

// Change the commit according to cfg
func changeCommit(commit string, cfg cfg.Cfgs) error {
    r := cfg.Git

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
            err = rebase(commit, cfg)
        case "checkout":
            err = checkout(commit, cfg)
    }

    // Check it worked
    if err != nil {
        switch r.CommitStrategy {
            // We can't do anything about a checkout failure
            case "checkout":
                return err
            // But we can for a rebase
            case "rebase":
                err = changeGood(err, true, cfg)
                if err != nil {
                    return err
                }
        }
    }

    return nil
}

// Merge/rebase the remote into the branch according to cfg
func reconcileRemote(cfg cfg.Cfgs) error {
    r := cfg.Git

    var err error

    // Perform the action
    switch r.RemoteStrategy {
        case "rebase":
            err = rebase(fmt.Sprintf("%s/%s", r.RemoteName, r.RemoteBranch), cfg)
        case "merge":
            err = merge(cfg)
    }

    // Check it worked
    err = changeGood(err, true, cfg)
    if err != nil {
        return err
    }

    return nil
}
