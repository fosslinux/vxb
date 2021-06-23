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

    // Create the masterdir
    // masterdirs use host arch
    err = vpkgs.CreateMasterdir(cfg.VpkgPath, cfg.HostArch, mountType, cfg.MountSize[mountType])
    if err != nil {
        return fmt.Errorf("%w while creating masterdir\n", err)
    }

    // Go!
    args := "pkg -N " + pkgname
    _, err = vpkgs.XbpsSrc(cfg.VpkgPath, cfg.HostArch, arch, args, true)
    if err != nil {
        return fmt.Errorf("%w building %s", err, ident)
    }

    // Remove the masterdir
    err = vpkgs.RemoveMasterdir(cfg.VpkgPath)

    return nil
}
