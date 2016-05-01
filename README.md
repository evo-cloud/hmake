# HyperMake

HyperMake helps you build projects without installing pre-requisites in your
local environment.
It uses containers to build projects, all pre-requisites can be pre-installed
inside the container.

A similar project is [drone](http://readme.drone.io) which is built to be a service.
While HyperMake is built as a handy tool with a few special features:

- Brings in the _target_ concept from traditional _GNU make_,
  targets can be defined in one HyperMake file or multiple,
  and targets has dependencies, can be built selectively.
- Concurrent builds, targets without explicit dependencies can be built concurrently.

## Getting Started

Download the binary from github release and place it in a folder which can be
searched in `PATH`.

Or if you know Go well enough

```
go get github.com/evo-cloud/hmake
```

Or if you want to build from source

```
# fetch all dependencies
gvt restore
# do a bootstrap
go build -o bin/hmake .
# do a full build
bin/hmake
```

_hmake_ expects a `HyperMake` file in root of the project,
and you can run `hmake` from any sub-directory inside the project,
the command will figure out project root by locating `HyperMake` file.

In any sub-directory, files called `*.hmake` can be included in `HyperMake` in
the root directory or any other `*.hmake` files.

## File Format

In `HyperMake` or `*.hmake`, define any of the following things:

- Targets: the target to build, including dependencies and commands.
- Settings: the settings applies to _hmake_.
- Includes: include more `*.hmake` files.

Here's the schema in example:

```yaml
---
format: hypermake.v0 # this indicates this is a HyperMake file

# project name and description
name: hmake
description: HyperMake builds your project without pre-requisites

# define targets
targets:
    hmake-linux-amd64:
        description: static linked hmake binary for Linux AMD64
        after: [vendor]
        watches:
            - '**/**/*.go'
        cmds:
            - env
            - >
                env GOOS=linux GOARCH=amd64
                go build -o bin/hmake-$GOOS-$GOARCH
                -a -tags 'static_build netgo' -installsuffix netgo
                -ldflags '-extldflags -static'
                .

    hmake-darwin-amd64:
        description: static linked hmake binary for Mac OS
        after: [vendor]
        watches:
            - '**/**/*.go'
        cmds:
            - env
            - >
                env GOOS=linux GOARCH=amd64
                go build -o bin/hmake-$GOOS-$GOARCH
                -a -tags 'static_build netgo' -installsuffix netgo
                -ldflags '-extldflags -static'
                .

    vendor:
        description: pull all vendor packages
        watches:
            - 'vendor/manifest'
        cmds:
            - 'apk update && apk add git'
            - 'go get github.com/FiloSottile/gvt'
            - 'gvt restore'
            - 'mkdir -p bin'

    all:
        after: [hmake-linux-amd64, hmake-darwin-amd64]

# settings shared across targets
settings:
    default-targets: [all]
    image: golang:1.6-alpine
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

#### State Directory

_hmake_ creates a state directory `$HMAKE_PROJECT_DIR/.hmake` to store logs and state files.
The output (stdout and stderr combined) of each target is stored in files `TARGET.log`.

## Command Usage

#### Usage

```
hmake [OPTIONS] [TARGETS]
```

#### Options

- `--parallel=N, -p N`: Set maximum number of targets executed in parallel, 0 for auto, -1 for unlimited
- `--rebuild-all, -R`: Force rebuild all needed targets
- `--rebuild TARGET, -r TARGET`: Force rebuild specified target, this can repeat
- `--json`: Dump execution events to stdout each encoded in single line json
- `--verbose, -v`: Show execution output to stderr for each target
- `--color|--no-color`: Explicitly specify print with color/no-color
- `--emoji|--no-emoji`: Explicitly specify print with emoji/no-emoji
- `--debug`: write a debug log `hmake.debug.log` in hmake state directory
