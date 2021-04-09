// SPDX-License-Identifier: BSD-2-Clause

package git

import (
    "github.com/fosslinux/vxb/vpkgs"
    "errors"
    "fmt"
)

func (r Repo) Changed(arch string, commits ...string) ([]string, error) {
    var err error
    errRet := []string{}

    // We only support:
    // 0 arguments: diff from current checkout to remote
    // 1 argument: diff to tip of remote
    // 2 arguments: diff between two specific commits
    var commita string
    var commitb string
    // I.e. if we were given no argument
    if len(commits) == 1 && commits[0] == "" {
        commita = "HEAD"
        commitb = fmt.Sprintf("%s/%s", r.RemoteName, r.RemoteBranch)
    } else if len(commits) == 1 {
        commita = commits[0]
        commitb = fmt.Sprintf("%s/%s", r.RemoteName, r.RemoteBranch)
    } else if len(commits) == 2 {
        commita = commits[0]
        commitb = commits[1]
    } else if len(commits) > 2 {
        return errRet, errors.New("Changed() only supports two commits max")
    }

    // If we are using 0-1 commits then we need to pull the remote into
    // master.
    if len(commits) == 0 || len(commits) == 1 {
        // First assume remotes are being used for 0-1 commits specified...
        // if not then error out
        if !r.WithRemote {
            return errRet, errors.New("Remotes must be enabled for zero or one arguments for git")
        }

        // Fetch
        err = r.fetch()
        if err != nil {
            return errRet, err
        }

        // Reconcile with the branch
        err = r.reconcileRemote()
        if err != nil {
            return errRet, err
        }
    }

    return r.changedAb(arch, commita, commitb)
}

func (r Repo) changedAb(arch string, commita string, commitb string) ([]string, error) {
    var err error
    errRet := []string{}

    // Perform a sanity check - commita should have NO not up-to-date packages
    // Only test this if we are not going from HEAD
    if commita != "HEAD" {
        err = r.changeCommit(commita)
        if err != nil {
            return errRet, err
        }
        ready, notReadyPkgs, err := vpkgs.Ready(arch, r.Path)
        if err != nil {
            return errRet, err
        }
        if !ready {
            fmt.Printf("%v\n", notReadyPkgs)
            return errRet, fmt.Errorf("%s (commit to go from) must NOT have any outdated packages (listed above)!")
        }
    }

    // Checkout commitb
    err = r.rebase(commitb)

    // Check for outdated packges
    _, outdated, err := vpkgs.Ready(arch, r.Path)
    if err != nil {
        return outdated, err
    }

    return outdated, nil
}
