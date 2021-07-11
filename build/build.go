package build

import (
    "github.com/fosslinux/vxb/vpkgs"
    "github.com/fosslinux/vxb/cfg"
    "fmt"
    str "strings"
)

// Specific wrapper command for building
func Build(ident string, cfg cfg.Cfgs) error {
    var err error

    splitIdent := str.Split(ident, "@")
    pkgname := splitIdent[0]
    arch := splitIdent[1]

    // Determine mount type to use
    mountType, exists := cfg.MountPkgs[pkgname]
    if !exists {
        mountType = cfg.MountDefault
    }

    // Create masterdir
    err = vpkgs.CreateMasterdir(mountType, cfg)
    if err != nil {
        return err
    }

    // Perform operation
    args := "pkg -N " + pkgname
    _, err = vpkgs.XbpsSrc(args, arch, mountType, true, cfg)
    if err != nil {
        // Attempt to remove masterdir
        vpkgs.RemoveMasterdir(cfg)
        return fmt.Errorf("%w building %s", err, ident)
    }

    // Remove masterdir
    err = vpkgs.RemoveMasterdir(cfg)
    if err != nil {
        return err
    }

    return nil
}
