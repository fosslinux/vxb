# vxb

'vxb' or the Void Xbps Builder is a simplistic build scheduler and subsequent
builder that intends to be used firstly in a short-term capacity by Void Linux
until a more suitable replacement presents itself, and by users and smaller
forks of Void Linux interested in building their own package repositories.

## Architecture

Starting at the beginning of the process, the inputs vxb takes is
either Git commits or package names. vxb can be used with or without Git
integration. When using Git commits, depending on the number of arguments, it
either:

1. 0 arguments: gets all outdated packages changed from current HEAD to
   remote.
2. 1 argument: gets all outdated packages changed from commit in argument to
   remote.
3. 2 argumens: gets all outdated packages between commita and b. Specified
   using commita...commitb.

When using package names, it simply takes those package names specified as
input. It also takes various other pieces of information, most notably the
architecture being built for.

vxb then enters the "graphing" phase. In this phase, it goes through some
recursive steps:

1. Loop through all of the package names (given by the Git process or from
   command line), and perform the following process on each:
2. Resolve the package name if it is a subpackage into its base package. (This
   is how we handle subpackages - they are not otherwise special to vxb. xbps-src
   is smart enough to figure out the architecture stuff for vxb).
3. Check if the package is already so-called "Ready". Ready means to vxb that it
   exists and is not out-of-date. If a package is ready, then we do not need to
   build it, it is not added to the graph, and all subsequent logic is
   unnessecary. We skip to the next package if it is ready.
4. Run dbulk-dump on the package, getting the hostmakedepends, makedepends and
   depends, grouping these into two categories; HostDepends, consisting of all
   packages in hostmakedepends, and Depends, consisting of all packages in
   makedepends and depends. These are then ensured that they have no duplicates.
5. For each of the "depends" packages, they are then added to the graph as a
   dependency of the original package, then steps 3-4 are repeated recursively
   for them. The process for "hostdepends" is identical, but the architecture is
   rewritten as the host architecture.
6. vxb now has a full graph of all packages it needs to build.
7. This is written to a .dot file a) for debugging b) cause it looks cool.

We can finally start the "building" phase. Currently, this only supports local
builds, and fails hard on a build failure, and has a number of other
limitations. The steps for this process are:

1. Begin with each of the source packages that we began with.
2. From each of these, traverse the graph depth-first. Build each package along
   the way. Because the graphs regularly overlap, we set packages as being
   "ready" once they are successfully built. Check that each package is not
   ready before building it.
3. If we have a failure, hard error out.
4. Done!

## Features

:heavy_check_mark: means implemented.
:x: means to be implemented.
:heavy_exclamation_mark: means it will not be implemented (non-feature).
:question: means it remains to be seen whether this shall be implemented.

| Feature                                                      | Status                   |
|--------------------------------------------------------------|--------------------------|
| Graphing packages                                            | :heavy_check_mark:       |
| Building packages through xbps-src                           | :heavy_check_mark:       |
| Building list of packages on command line                    | :heavy_check_mark:       |
| Building packages using Git                                  | :heavy_check_mark:       |
| Building packages locally                                    | :heavy_check_mark:       |
| Building packages in Docker                                  | :x:                      |
| Building packages in Nomad                                   | :x:                      |
| Subpackage support                                           | :heavy_check_mark:       |
| -32bit package support                                       | :x:                      |
| Ability to set to build *all* packages (official repo style) | :x:                      |
| Building different graph paths on failure                    | :x:                      |
| Web UI                                                       | :x:                      |
| Visual representation of graph                               | :heavy_check_mark:       |
| Configuration system                                         | :heavy_check_mark:       |
| Global configuration from Web UI                             | :heavy_exclamation_mark: |
| Mounting masterdir on tmpfs/zram                             | :x:                      |
| Package signing                                              | :question:               |
| Repo cleaning                                                | :question:               |
