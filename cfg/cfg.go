// SPDX-License-Identifier: BSD-2-Clause

package cfg

import (
    getoptions "github.com/DavidGamba/go-getoptions"
    "github.com/ryanuber/go-glob"
    "github.com/go-ini/ini"
    "golang.org/x/sys/unix"
    str "strings"
    "os"
    "fmt"
)

// Configuration struct
type Cfgs struct {
    // Path to void-packges checkout
    VpkgPath string
    // "subrepo" i.e. -r argument
    SubRepos map[string]string
    // Architecture to build for
    Arch string
    // Packages to build (specified on cmdline)
    SPkgNames string
    // Host architecture
    HostArch string
    // Path to configuration file
    ConfPath string
    // Modifications are being made from upstream void-packages
    Mods bool
    // Information about the system
    SysInfo *unix.Utsname
    // As a string the machine type
    MachineType string
    // Default mount type
    MountDefault string
    // Specific package mount types
    MountPkgs map[string]string
    // Sizes of different types of mounts
    MountSize map[string]string

    // Other structures
    // All of the git configuration
    Git *Repo
    // Option parsing
    Opt *getoptions.GetOpt
    // Configuration file parsing
    cfgf *ini.File
}

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

// Basic list of valid architectures
var validArchs = []string{"aarch64", "armv5tel", "armv6l", "armv7l", "i686",
    "mips-musl", "mipsel-musl", "mipselhf-musl", "mipshf-musl", "ppc",
    "ppc64", "ppc64le", "ppcle", "x86_64"}

// Create the options structure
func (cfg *Cfgs) InitOpt() {
    // Add -musl variants to validArchs
    for _, arch := range validArchs {
        if !str.HasSuffix(arch, "-musl") {
            validArchs = append(validArchs, arch + "-musl")
        }
    }

    cfg.Opt = getoptions.New()
    cfg.Opt.SetMode(getoptions.Bundling)
    // Create the git structure
    cfgGit := Repo{}
    cfg.Git = &cfgGit
}

// Add options
func (cfg *Cfgs) AddOpts() {
    opt := cfg.Opt

    opt.Bool("help", false, opt.Alias("h"))
    opt.StringVar(&cfg.VpkgPath, "vpkg", "", opt.Alias("v"),
        opt.Description("Path to void-packages checkout."))
    opt.StringVar(&cfg.Arch, "arch", "", opt.Required(), opt.Alias("a"),
        opt.Description("The architecture to build for."))
    opt.StringVar(&cfg.SPkgNames, "pkgname", "", opt.Alias("p"),
        opt.Description("The package(s) to build."))
    opt.StringVarOptional(&cfg.Git.Commits, "git", "", opt.Alias("g"),
        opt.Description("Git commits to update between."))
    opt.StringVar(&cfg.HostArch, "hostarch", "", opt.Alias("m"),
        opt.Description("The host architecture."))
    opt.StringVar(&cfg.ConfPath, "conf", "conf.ini", opt.Alias("c"),
        opt.Description("Configuration file path."))
    opt.BoolVar(&cfg.Mods, "mods", false, opt.Alias("d"),
        opt.Description("Modifications are made from upstream void-packages."))
}

// Act on options
func (cfg *Cfgs) ActOpts(remaining []string, err error) {
    // If we errored show the error and a help
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
        fmt.Fprintf(os.Stderr, cfg.Opt.Help(getoptions.HelpSynopsis))
        os.Exit(1)
    }

    // They asked for help, give them hep
    if cfg.Opt.Called("help") {
        fmt.Fprintf(os.Stderr, cfg.Opt.Help())
        os.Exit(0)
    }

    // Warn for unhandled arguments
    if len(remaining) != 0 {
        fmt.Fprintf(os.Stderr, "WARN: Unhandled arguments: %v\n", remaining)
    }
}

// Check the config file exists
func (cfg *Cfgs) haveCfgFile() bool {
    // Stat the supposed config file
    _, exists := os.Stat(cfg.ConfPath)
    if os.IsNotExist(exists) {
        if cfg.ConfPath == "conf.ini" {
            // If it is the default, there is no config file
            return false
        } else {
            // We were given a bad config file
            fmt.Fprintf(os.Stderr, "ERROR: Cannot open config file %s!\n", cfg.ConfPath)
            os.Exit(1)
        }
    }

    return true
}

// Initialize config file struct
// Returns if there is a config file
func (cfg *Cfgs) InitCfg() (bool, error) {
    // Check if it exists
    if !cfg.haveCfgFile() {
        return false, nil
    }

    // Open it
    var err error
    cfg.cfgf, err = ini.Load(cfg.ConfPath)
    if err != nil {
        return true, err
    }

    return true, nil
}

// Parse the default mount type
func (cfg *Cfgs) parseMountDefault() {
    cfg.MountDefault = cfg.cfgf.Section("mount").Key("default").String()
    if cfg.MountDefault == "" {
        cfg.MountDefault = "none"
    // Valid: none, tmpfs, zram, zram-std
    } else if cfg.MountDefault != "none" &&
                cfg.MountDefault != "tmpfs" &&
                cfg.MountDefault != "zram" &&
                cfg.MountDefault != "zram-zstd" {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid default mount type (valid: none, tmpfs, zram, zram-zstd.\n");
        os.Exit(1)
    }
}

// Parse the tmpfs size
func (cfg *Cfgs) parseMountSize() {
    cfg.MountSize = make(map[string]string)
    cfg.MountSize["tmpfs"] = cfg.cfgf.Section("mount").Key("tmpfs_size").String()
    cfg.MountSize["zram"] = cfg.cfgf.Section("mount").Key("zram_size").String()
    cfg.MountSize["zram-zstd"] = cfg.cfgf.Section("mount").Key("zram_zstd_size").String()
}

// Parse the mount.pkgs section
func (cfg *Cfgs) parseMountPkgs() {
    cfg.MountPkgs = cfg.cfgf.Section("mount.pkgs").KeysHash()

    // Ensure each of the pkgs given is valid as is the type
    for pkgName, mountType := range cfg.MountPkgs {
         _, err := os.Stat(cfg.VpkgPath + "/srcpkgs/" + pkgName)
         if os.IsNotExist(err) {
             fmt.Fprintf(os.Stderr, "WARN: Package %s does not exist, which was attempted to use %s!", pkgName, mountType)
         }
         if mountType != "none" &&
            mountType != "tmpfs" &&
            mountType != "zram" &&
            mountType != "zram-zstd" {
            fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid package mount type (used for %s).\n", mountType, pkgName)
            os.Exit(1)
        }
    }
}

// Parse subrepos
func (cfg *Cfgs) parseSubrepo() {
    original := cfg.cfgf.Section("vpkg.subrepo").KeysHash()
    // Init map
    cfg.SubRepos = make(map[string]string)

    // Expand globs + validate
    for pattern, value := range original {
        found := false
        // Test pattern against each arch
        for _, arch := range validArchs {
            if glob.Glob(pattern, arch) {
                found = true
                cfg.SubRepos[arch] = value
            }
        }
        // Error out if nothing was found
        if !found {
            fmt.Fprintf(os.Stderr, "ERROR: No architecture matches to %s.\n", pattern)
            os.Exit(1)
        }
    }
}

// Parse the config file
func (cfg *Cfgs) ParseCfg() error {
    var err error

    // Path to void-packages
    if !cfg.Opt.Called("vpkg") {
        cfg.VpkgPath = cfg.cfgf.Section("vpkg").Key("path").String()
    }

    err = cfg.parseHostArch()
    if err != nil {
        return err
    }

    cfg.parseSubrepo()

    cfg.validateArch()

    // Modifications
    if !cfg.Opt.Called("mods") {
        var err error
        cfg.Mods, err = cfg.cfgf.Section("").Key("mods").Bool()
        if err != nil && cfg.cfgf.Section("").Key("mods").String() == "" {
            // Warn on auto-detection of this
            fmt.Fprintf(os.Stderr, "WARN: Assuming there are no local modifications to void-packages.")
            cfg.Mods = false
        }
    }

    cfg.parseMountDefault()
    cfg.parseMountPkgs()
    cfg.parseMountSize()
    cfg.validateMountSizes()

    return nil
}

// Parse hostarch
func (cfg *Cfgs) parseHostArch() error {
    // Host architecture
    if !cfg.Opt.Called("hostarch") {
        cfg.HostArch = cfg.cfgf.Section("vpkg").Key("host_arch").String()
        // If there is still nothing, use the default logic
        if cfg.HostArch == "" {
            err := unix.Uname(cfg.SysInfo)
            if err != nil {
                return fmt.Errorf("Error %w getting information about system", err)
            }
            cfg.HostArch = string(cfg.SysInfo.Machine[:])
        }
    }
    // Check hostArch
    hostFound := false
    for _, tArch := range validArchs {
        if tArch == cfg.HostArch {
            hostFound = true
            break
        }
    }
    if !hostFound {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid architecture.\n", cfg.HostArch)
        os.Exit(1)
    }

    return nil
}

// Parse whether git is enabled
func (cfg *Cfgs) parseGitEnable() {
    var err error
    cfg.Git.Enable, err = cfg.cfgf.Section("git").Key("enable").Bool()
    if err != nil {
        cfg.Git.Enable = false
    }
}

// Parse if a remote is being used
func (cfg *Cfgs) parseGitWithRemote() {
    var err error
    cfg.Git.WithRemote, err = cfg.cfgf.Section("git").Key("with_remote").Bool()
    // Must be set
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: Not (validly) specified if git remotes are being used.")
        os.Exit(1)
    }
}

// Parse the name of the remote being used
func (cfg *Cfgs) parseGitRemoteName() {
    cfg.Git.RemoteName = cfg.cfgf.Section("git").Key("remote_name").String()
    if cfg.Git.RemoteName == "" {
        fmt.Fprintf(os.Stderr, "ERROR: Git remotes are used but no remote name was specified in config fil.")
        os.Exit(1)
    }
}

// Parse the branch of the remote being used
func (cfg *Cfgs) parseGitRemoteBranch() {
    cfg.Git.RemoteBranch = cfg.cfgf.Section("git").Key("remote_branch").String()
    if cfg.Git.RemoteBranch == "" {
        fmt.Fprintf(os.Stderr, "ERROR: Git remotes are used but no remote name was specified in config fil.")
        os.Exit(1)
    }
}

// Parse the strategy to use to pull in changes from remote
func (cfg *Cfgs) parseGitRemoteStrategy() {
    cfg.Git.RemoteStrategy = cfg.cfgf.Section("git").Key("remote_strategy").String()
    if cfg.Git.RemoteStrategy == "" {
        // If mods, rebase, otherwise ff
        if cfg.Mods {
            cfg.Git.RemoteStrategy = "rebase"
        } else {
            cfg.Git.RemoteStrategy = "ff"
        }
    // Valid: ff, rebase, merge
    } else if cfg.Git.RemoteStrategy != "ff" &&
                cfg.Git.RemoteStrategy != "rebase" &&
                cfg.Git.RemoteStrategy != "merge" {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid remote strategy (valid: ff, rebase, merge).\n", cfg.Git.RemoteStrategy)
        os.Exit(1)
    }
}

// Parse the strategy to use to change commits
func (cfg *Cfgs) parseGitCommitStrategy() {
    cfg.Git.CommitStrategy = cfg.cfgf.Section("git").Key("commit_strategy").String()
    if cfg.Git.CommitStrategy == "" {
        // If we have modifications, rebase, otherwise checkout
        if cfg.Mods {
            cfg.Git.CommitStrategy = "rebase"
        } else {
            cfg.Git.CommitStrategy = "checkout"
        }
    // Valid: rebase, checkout
    } else if cfg.Git.CommitStrategy != "rebase" && cfg.Git.CommitStrategy != "checkout" {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid commit strategy (valid: rebase, checkout).\n", cfg.Git.CommitStrategy)
        os.Exit(1)
    }
}

// Parse what should happen hen merge/rebase/checkout fails, should we die or
// drop to shell?
func (cfg *Cfgs) parseGitChangeFail() {
    cfg.Git.ChangeFail = cfg.cfgf.Section("git").Key("fail").String()
    if cfg.Git.ChangeFail == "" {
        cfg.Git.ChangeFail = "shell"
    // Valid: shell, die
    } else if cfg.Git.ChangeFail != "shell" && cfg.Git.ChangeFail != "die" {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a vaild failure option (valid: die, shell).\n", cfg.Git.ChangeFail)
        os.Exit(1)
    }
}

// Parse the git part of the config file
func (cfg *Cfgs) ParseGitCfg() {
    cfg.parseGitEnable()

    if cfg.Git.Enable {
        cfg.Git.Path = cfg.VpkgPath
        // Branch name
        cfg.Git.Branch = cfg.cfgf.Section("git").Key("branch").String()
        cfg.parseGitWithRemote()
        // Are we using a remote?
        if cfg.Git.WithRemote {
            cfg.parseGitRemoteName()
            cfg.parseGitRemoteBranch()
            cfg.parseGitRemoteStrategy()
        }

        cfg.parseGitCommitStrategy()
        cfg.parseGitChangeFail()
    }
}

// Evaluate automatic -musl extension
func (cfg *Cfgs) EvalAutoMuslExt() {
    // If the architecture is -musl and the host was not manually set, then
    // the host should also be -musl.
    hostManual := !cfg.Opt.Called("hostarch") || cfg.SysInfo != nil
    if str.HasSuffix(cfg.Arch, "-musl") && hostManual {
        cfg.HostArch += "-musl"
    }
}

// Validate that a VpkgPath was given
func (cfg *Cfgs) ValidVpkgPath() {
    if cfg.VpkgPath == "" {
        fmt.Fprintf(os.Stderr, "ERROR: No path to void-packages was given.\n")
        os.Exit(1)
    }
}

// Validate that git commits and git enabled make sense
func (cfg *Cfgs) ValidGitEnabled() {
    // If git commits were given on command line options, git must be enabled
    if cfg.Git.Commits != "" && !cfg.Git.Enable {
        fmt.Fprintf(os.Stderr, "ERROR: Specified git on command line but git is disabled.")
        os.Exit(1)
    }
}

// Validate arch and hostArch are known
func (cfg *Cfgs) validateArch() {
    // Check arch
    archFound := false
    for _, tArch := range validArchs {
        if tArch == cfg.Arch {
            archFound = true
            break
        }
    }
    if !archFound {
        fmt.Fprintf(os.Stderr, "ERROR: %s is not a valid architecture.\n", cfg.Arch)
        os.Exit(1)
    }
}

// Validate that we have sizes for types of mounts we use
func (cfg *Cfgs) validateMountSizes() {
    // Checking default
    if cfg.MountSize[cfg.MountDefault] == "" &&
        cfg.MountDefault != "none" {
        fmt.Fprintf(os.Stderr, "ERROR: %s is the default mount type but does not have a size set.\n", cfg.MountDefault)
        os.Exit(1)
    }

    // Checking packages
    for pkgName, mountType := range cfg.MountPkgs {
        if cfg.MountSize[mountType] == "" &&
            mountType != "none" {
            fmt.Fprintf(os.Stderr, "ERROR: %s is the mount type used for %s but does not have a size set.\n", mountType, pkgName)
            os.Exit(1)
        }
    }
}

// We must be given *something* to do
func (cfg *Cfgs) validDo() {
    // Either a package must be given or git commit must be given
    if !cfg.Opt.Called("pkgname") && !cfg.Opt.Called("git") {
        fmt.Fprintf(os.Stderr, "ERROR: Either packages to build or git must be specified.")
        os.Exit(1)
    }
}
