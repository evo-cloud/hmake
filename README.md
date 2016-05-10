[![Build Status](https://travis-ci.org/evo-cloud/hmake.svg?branch=master)](https://travis-ci.org/evo-cloud/hmake)

# HyperMake

- Are you feeling bored and upset with spending days on preparing development
environment to build some project from source?
- Are you tired of writing a long list of building steps when you ships your work
to others?
- Are you crazy struggling with environment issues and try to make something work?

Forget about environment setup,
HyperMake helps you build projects instantly and consistently without installing
pre-requisites in your local environment.
It uses containers to build projects, all pre-requisites are pre-installed
inside the container.

To build _ANY_ HyperMake project, the _ONLY_ software needed are:

- A running [docker](https://www.docker.com)
- `hmake` executable in `PATH`

A similar project is [drone](http://readme.drone.io) which is built to be a service.
While HyperMake is built as a handy tool with a few special features:

- Brings in the _target_ concept from traditional _GNU make_,
  targets can be defined in one HyperMake file or multiple,
  and targets have dependencies, can be built selectively.
- Concurrent builds, targets without explicit dependencies can be built concurrently.

## Getting Started

#### The Simplest Way

Download the binary from github release:

```
curl -s https://github.com/evo-cloud/hmake/releases/download/v1.0.0rc1/hmake-1.0.0rc1-linux-amd64.tar.gz | sudo tar -C /usr/local/bin -zx
chmod a+rx /usr/local/bin/hmake
```

#### With Go installed

```
go get github.com/evo-cloud/hmake
```

#### Build from source

```
git clone https://github.com/evo-cloud/hmake
cd hmake
# first do a bootstrap
go get
go install
# do a full build
hmake
```

Note: if you want to use `hmake` to do a full build, make sure [docker](https://www.docker.com) is running.

#### You want tests?

After build, `hmake test`.
If you want run test with coverage, `hmake cover`.

## Setup Development Environment

_In fact with `hmake` installed, you don't this local development environment._

Install
- [gvt](github.com/FiloSottile/gvt)
- [ginkgo](https://github.com/onsi/ginkgo)
- [gomega](https://github.com/onsi/gomega)

```bash
go get https://github.com/FiloSottile/gvt
go get https://github.com/onsi/ginkgo/ginkgo
go get https://github.com/onsi/gomega
gvt restore
ginkgo test ./test
# or
go test ./test
go test -coverprofile=cover.out -coverpkg=./project ./test
```

## How it works

_hmake_ expects a `HyperMake` file in root of the project,
and you can run `hmake` from any sub-directory inside the project,
the command will figure out project root by locating `HyperMake` file.

From the top-level `HyperMake` file, use `includes` section to include any
`*.hmake` files inside project tree.
It can't access any files outside project tree.

By default, it will search for `.hmakerc` files inside project directories, and
load them in the order from parent to child folders.

### File Format

In `HyperMake` or `*.hmake`, define any of the following things:

- Format: the format presents the current file, should always be `hypermake.v0`;
- Name and description: only defined in top-level `HyperMake` file;
- Targets: the target to build, including dependencies and commands;
- Settings: the settings applies to _hmake_;
- Includes: include more `*.hmake` files.

Here's the schema in example (this is actually the `HyperMake` file of `hmake` project):

```yaml
---
format: hypermake.v0 # this indicates this is a HyperMake file

# project name and description
name: hmake
description: HyperMake builds your project without pre-requisites

# define targets
targets:
    builder:
        description: build the docker image including toolchain
        build: builder/Dockerfile
        image: hmake-builder:latest
        watches:
            - builder

    hmake-linux-amd64:
        description: static linked hmake binary for Linux AMD64
        after:
            - vendor
        watches:
            - '**/**/*.go'
        cmds:
            - ./build.sh linux amd64

    hmake-darwin-amd64:
        description: static linked hmake binary for Mac OS
        after:
            - vendor
        watches:
            - '**/**/*.go'
        cmds:
            - ./build.sh darwin amd64

    vendor:
        description: pull all vendor packages
        after:
            - builder
        watches:
            - vendor/manifest
        envs:
            - HMAKE_VER_SUFFIX
            - HMAKE_RELEASE            
        cmds:
            - gvt restore
            - mkdir -p bin
            - ./build.sh genver

    test:
        description: run tests
        after:
            - vendor
        watches:
            - '**/**/*.go'
            - test
        cmds:
            - ginkgo ./test

    cover:
        description: run tests with coverage
        after:
            - vendor
        watches:
            - '**/**/*.go'
            - test
        cmds:
            - >
                go test -coverprofile cover.out
                -coverpkg ./project
                ./test

    all:
        description: the default make target
        after:
            - hmake-linux-amd64
            - hmake-darwin-amd64

# settings shared across targets
settings:
    default-targets:
        - all
    docker:
        image: hmake-builder:latest
        src-volume: /go/src/github.com/evo-cloud/hmake

includes:
    - build/**/**/*.hmake
```

#### Dependencies

Dependencies can be specified using:

- `after`: the target is executed when the depended tasks succeed or are skipped
- `before`: the target must succeed or skip before the specified tasks get executed.

In most cases, `after` is enough in a single file.
`before` is mostly used to inject dependencies in the files included.

#### Include files

In `includes` section, specify files to be included.
The files included can provide more targets and also override settings.

#### Pre-defined Environment Variables

- `HMAKE_PROJECT_NAME`: the name of the project
- `HMAKE_PROJECT_DIR`: the directory containing `HyperMake` (aka. project root)
- `HMAKE_PROJECT_FILE`: the full path to `HyperMake`
- `HMAKE_WORK_DIR`: `$HMAKE_PROJECT_DIR/.hmake`
- `HMAKE_LAUNCH_PATH`: the relative path under `$HMAKE_PROJECT_DIR` where `hmake` launches
- `HMAKE_REQUIRED_TARGETS`: the names of targets explicitly required from command line, separate by space
- `HMAKE_TARGET`: the name of the target currently in execution
- `HMAKE_VERSION`: version of _hmake_
- `HMAKE_OS`: operating system
- `HMAKE_ARCH`: CPU architecture

#### Global Setting Properties

- `default-targets`: a list of targets to build when no targets are specified in `hmake` command
- `exec-driver`: the name of driver which parses properties in target and executes the target,
  the default value is `docker`, and supported drivers are `docker` and `shell`.
  This property can also be specified in target instead of global `settings` section.

#### Common Properties in Target

- `description`: description of the target
- `before`: a list of names of targets which can only execute after this target
- `after`: a list of names of targets on which this targets depends
- `exec-driver`: same as in `settings` section, but only specify the driver for this target
- `envs`: a list of environment variables (the form `NAME=VALUE`) to be used for execution
- `script`: a multi-line string represents a full script to execute for the target
- `cmds`: when `script` is not specified, this is a list of commands to execute for the target
- `watches`: a list of path/filenames (wildcard supported) whose mtime will be checked to determine if the target is out-of-date,
  without specifying this property, the target is always executed (the `.PHONY` target in `make`).

Other properties are driver specific, and will be parsed by driver.

#### State Directory

_hmake_ creates a state directory `$HMAKE_PROJECT_DIR/.hmake` to store logs and state files.
The output (stdout and stderr combined) of each target is stored in files `TARGET.log`.
Debug log (with `--debug`) is stored as `hmake.debug.log`.
Summary file is stored as `hmake.summary.json`.

### Execution Drivers

#### The `shell` driver

This is simplest driver which inteprets `script` or `cmds` as shell script/commands.
The list of `cmds` will be merged as a shell script.
And the intepreter is `/bin/sh`.
`set -e` is inserted as the first line to fail-fast.

#### The `docker` driver

This driver generates the same script as `shell` driver but run it inside a docker container.
The following properties are supported:

- `build`: path to `Dockerfile`, when specified, this target builds a docker image first.
   `image` property specifies the image name and tag.
   It's strongly recommended to put `Dockerfile` and any related files to `watches` list.
- `build-from`: the build path for `docker build`.
  Without this property, the build path is derived from path of `Dockerfile` specified in `build`.
- `image`: with `build` it's the image name and tag to build,
  without `build`, it's the image used to create the container.
- `src-volume`: the full path inside container where project root is mapped to.
  Default is `/root/src`.
- `expose-docker`: when set `true`, expose the host docker server connectivity into container to allow
  docker client run from inside the container.
  This is very useful when docker is required for build but to avoid problematic docker-in-docker.
- `envs`: list environment variables passed to container, can be `NAME=VALUE` or `NAME`.
- `env-files`: list of files providing environment variables, see `--env-files` of `docker run`
- `privileged`: run container in privileged mode, default is `false`
- `net`: when specified, only allowed value is `host`, when specified, run container with `--net=host --uts=host`
- `user`: passed to `docker run --user...`, by default, current `uid:gid` are passed
  It must be explicitly specified `root` if the script is executed as root inside container.
  When a non-root user is explicitly specified, all group IDs are passed using `--group-add`.
- `groups`: explicitly specify group IDs to pass into container, instead of passing all of them.
- `volumes`: a list of volume mappings passed to `-v` option of `docker run`.

The following properties maps to `docker run` options:

- `cap-add`, `cap-drop`
- `devices`
- `hosts`: mapped to `--add-host`
- `dns`, `dns-opts`, `dns-search`
- `blkio-weight`, `blkio-weight-devices`
- `device-read-bps`, `device-write-bps`, `device-read-iops`, `device-write-iops`
- `cpu-shares`, `cpu-period`, `cpu-quota`, `cpuset-cpus`, `cpuset-mems`
- `kernel-memory`, `memory`, `memory-swap`, `memory-swappiness`, `shm-size`

All above properties can also be specified in global `settings` under `docker` section:

```yaml
settings:
    docker:
        property: value
```

##### About volume mapping

By default the current project root is mapped into container at `src-volume`,
default value is `/root/src`.
And it's also the current working directory when script starts.
As the script is a shell script, the executable `/bin/sh` must be present in the container.

##### About user

By default `hmake` uses current user (NOT root) to run inside container,
which make sure any file change has the same permission as the environment outside.
If root is required, it can be explicitly specified `user: root`,
however, all files created inside container will show up being owned by `root` outside,
and you may end up seeing some error messages like `permission denied` when you do something later.

## Command Usage

#### Usage

```
hmake [OPTIONS] [TARGETS]
```

#### Options

- `--chdir=PATH, -C PATH`: Chdir to specified PATH first before doing anything
- `--include=FILE, -I FILE`: Include additional files (must be relative path under project root), can be specified multiple times
- `--define=key=value, -D key=value`: Define property in global `settings` section, `key` may include `.` to specify the hierarchy
- `--parallel=N, -p N`: Set maximum number of targets executed in parallel, 0 for auto, -1 for unlimited
- `--rebuild-all, -R`: Force rebuild all needed targets
- `--rebuild TARGET, -r TARGET`: Force rebuild specified target, this can repeat
- `--skip TARGET, -S TARGET`: Skip specified target (mark as Skipped), this can repeat
- `--json`: Dump execution events to stdout each encoded in single line json
- `--summary, -s`: Show execution summary before exit
- `--verbose, -v`: Show execution output to stderr for each target
- `--rcfile|--no-rcfile`: Load .hmakerc inside project directories, default is true
- `--color|--no-color`: Explicitly specify print with color/no-color
- `--emoji|--no-emoji`: Explicitly specify print with emoji/no-emoji
- `--debug`: Write a debug log `hmake.debug.log` in hmake state directory
- `--show-summary`: When specified, print previous execution summary and exit
- `--targets`: When specified, print list of target names and exit
- `--dryrun`: When specified, run targets as normal but without invoking execution drivers (simply mark task Success)
- `--version`: When specified, print version and exit

## Supported Platform and Software

- docker 1.9 and above (1.9 - 1.11 tested)
- Linux (Ubuntu 14.04 tested)
- Mac OS X 10.9 and above (10.9, 10.11 tested)

## License

MIT
