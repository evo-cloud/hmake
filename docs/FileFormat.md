# HyperMake File Format

File `HyperMake` must be present in root directory of the project tree. Command
`hmake` can be invoked in any sub-directories inside the project tree and it will
locate the root of project by looking up `HyperMake`.
Additional files must be named as `*.hmake` for being referenced in `includes`
section.
All these files share the same format.

In `HyperMake` or `*.hmake`, define the following things:

- Format: the format presents the current file, should always be `hypermake.v0`;
- Name and description: only defined in top-level `HyperMake` file;
- Targets: the target to build, including dependencies and commands;
- Settings: the settings applies to _hmake_ and should be merged into a global view;
- Local Settings: the settings only apply to current `HyperMake` or `.hmake` file;
- Includes: include more `*.hmake` files.

Here's the schema in example (this is from the `HyperMake` file of `hmake` project):

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
        build: builder
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
        env:
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

# same as settings, but only apply to targets in the same file
local:
    key: value

includes:
    - build/**/**/*.hmake
```

## Format

The format of this `YAML` file is indicated by `format` property which is
mandatory and the current acceptable value is `hypermake.v0`

## Name and Description

These are optional properties, while it's recommended `name` should be provided
as project name.

## Targets

The property `targets` defines a dictionary of named targets.
A target is a set of properties to define what to do (a script or a list of commands).
Usually, it defines

- The environment to execute the script/commands,
  like docker image, workdir, and options for `docker run`;
- The dependencies between targets, using `before` and `after` properties;
- Watch of files to decide whether the target should be rebuilt.

#### Common Properties in Target

- `description`: description of the target;
- `before`: a list of names of targets which can only execute after this target;
- `after`: a list of names of targets on which this targets depends;
- `workdir`: the current working directory for commands in the target,
  relative to project root;
- `watches`: a list of path/filenames (wildcard supported) whose mtime will be
  checked to determine if the target is out-of-date, without specifying this
  property, the target is automatically skipped if the last execution was successful
  and all dependencies are skipped;
- `always`: always build the target regardless of last execution state and results
  of all dependencies (the `.PHONY` target in `make`);

Other properties are specific to execution driver which executes the target.
The currently supported execution driver is `docker`, please read
[Docker Driver](DockerDriver.md) for details.

#### Dependencies

Dependencies are specified using:

- `after`: the target is executed when the depended tasks succeed or are skipped
- `before`: the target must succeed or skip before the specified tasks get executed.

A _skipped_ target means there's nothing to do with the target (no commands or
it's still up-to-date). It can be an equivalent to _success_.

In most cases, `after` is enough. `before` is often used to inject dependencies.

#### Matching targets names with wildcards

The places (`before`, `after`, `-r`, `-S`, command line targets, etc) requiring
target names accept wildcards:

- Wildcards used in file names: `*`, `?`, `\` and `[chars]`, they are matched using `filepath.Match`
- Regular Expression: the name starts and ends with `/`

#### Pre-defined Environment Variables

- `HMAKE_PROJECT_NAME`: the name of the project
- `HMAKE_PROJECT_DIR`: the directory containing `HyperMake` (aka. project root)
- `HMAKE_PROJECT_FILE`: the full path to `HyperMake`
- `HMAKE_WORK_DIR`: `$HMAKE_PROJECT_DIR/.hmake`
- `HMAKE_LAUNCH_PATH`: the relative path under `$HMAKE_PROJECT_DIR` where `hmake` launches
- `HMAKE_REQUIRED_TARGETS`: the names of targets explicitly required from command line, separate by space
- `HMAKE_TARGET`: the name of the target currently in execution
- `HMAKE_TARGET_DIR`: the relative path to directory containing the file which defines the target
- `HMAKE_VERSION`: version of _hmake_
- `HMAKE_OS`: operating system
- `HMAKE_ARCH`: CPU architecture

## Include Files

In `includes` section, specify files to be included.
The files included can provide more targets and also override settings.

Any path used in `HyperMake` or `*.hmake` files are relative to current file.
When a target gets executed, the default working directory is where the file
defining the target exists.

## Settings

In `settings` section, the hierarchical dictionary is used to provide
global settings. According to the order of `*.hmake` files loaded, the file loaded
latter overrides the settings in the former loaded files.
In `local` section, the settings are only applied to current file.
And the properties defined in target overrides everything.

Here's the order _hmake_ looks a setting by name:

- From target's properties;
- From `local`;
- From `settings` in the reversed order of files being loaded.

#### Pre-defined Setting Properties

- `default-targets`: a list of targets to build when no targets are specified
  in `hmake` command;
- `docker`: a set of [docker](DockerDriver.md) specific properties which defines
   default values for targets.

## Local Customization

After loading `HyperMake` and `*.hmake` files, _hmake_ also looks up `.hmakerc`
files from current directory up to root directory of the project and load them
in the order from root directory down to the current directory.
The `.hmakerc` has the same format as `HyperMake` and `*.hmake` files and is
used to override settings and inject targets to satisfy the special needs of
local development environment.
`.hmakerc` should be included in `.gitignore` file.
