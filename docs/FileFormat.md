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

    vendor:
        description: pull all vendor packages
        after:
            - builder
        watches:
            - 'vendor/manifest'
        cmds:
            - gvt restore

    # target hmake-linux- will be expanded to three target architectures
    hmake-linux-[arch:amd64,arm,arm64]:
        description: static linked hmake binary for linux-$[arch]
        after:
            - vendor
        watches:
            - '**/**/*.go'
            - build.sh
        cmds:
            - ./build.sh linux $[arch]
        artifacts:
            - bin/linux/$[arch]/hmake
            - bin/hmake-linux-$[arch].tar.gz
            - bin/hmake-linux-$[arch].tar.gz.sha256sum

    hmake-darwin-amd64:
        description: static linked hmake binary for Mac OS
        after:
            - vendor
        watches:
            - '**/**/*.go'
            - build.sh
        cmds:
            - ./build.sh darwin amd64
        artifacts:
            - bin/darwin/amd64/hmake
            - bin/hmake-darwin-amd64.tar.gz
            - bin/hmake-darwin-amd64.tar.gz.sha256sum

    hmake-windows-amd64:
        description: static linked hmake binary for Windows
        after:
            - vendor
        watches:
            - '**/**/*.go'
            - build.sh
        cmds:
            - ./build.sh windows amd64
        artifacts:
            - bin/windows/amd64/hmake.exe
            - bin/hmake-windows-amd64.zip
            - bin/hmake-windows-amd64.zip.sha256sum

    site:
        description: generate document site
        after:
            - builder
        watches:
            - site/gh-pages/config.toml
            - site/gh-pages/themes/**/**/*
            - site/gh-pages/static/**/**/*
            - README.md
            - docs/**/**/*
            - examples/*/README.md
            - build.sh
        cmds:
            - ./build.sh gensite

    checkfmt:
        description: check code format
        after:
            - builder
        always: true
        cmds:
            - ./build.sh checkfmt

    lint:
        description: check code using metalint
        after:
            - builder
        always: true
        cmds:
            - ./build.sh lint

    check:
        description: check source code
        after:
            - checkfmt
            - lint

    test:
        description: run tests
        after:
            - vendor
        always: true
        cmds:
            - ginkgo ./test

    cover:
        description: run tests with coverage
        after:
            - vendor
        always: true
        cmds:
            - >
                go test -coverprofile cover.out
                -coverpkg ./project
                ./test

    e2e:
        description: end-to-end tests
        after:
            - vendor
        expose-docker: true
        always: true
        cmds:
            - ginkgo ./test/e2e

    all:
        description: the default make target
        after:
            - hmake-linux-amd64
            - hmake-darwin-amd64
            - hmake-windows-amd64

# define some special targets which can be used as commands
commands:
    echo:
        description: simple echo command
        cmds:
            - 'echo $@'

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

## Commands

Commands are special type of targets, when used, it must be the first non-option
argument in command line.
E.g. invoking `hmake` like `hmake name arg1 arg2`, if `name` is defined
in `commands`, the rest of arguments in the command line is passed as arguments
to command target `name`.

Commands are targets, with a few restrictions:

- Commands should not be dependency of others. So `before` is not allowed, and
  other targets/commands should not `after` a command;
- Command can only be used as the first non-option argument in _hmake_ command line
  which turns on _command mode_. In the case `hmake target1 cmd1`, it refuses to
  run because `cmd1` is a command but not come first.

#### Common Properties in Target

- `description`: description of the target;
- `before`: a list of names of targets which can only execute after this target;
- `after`: a list of names of targets on which this targets depends;
- `workdir`: the current working directory for commands in the target,
  relative to `.hmake` file defining the target;
  if it's absolute (starting with `/`), it's relative to project root;
  by default, it's the current directory containing the `.hmake` file.
- `watches`: a list of path/filenames (wildcard supported) whose mtime will be
  checked to determine if the target is out-of-date, without specifying this
  property, the target is automatically skipped if the last execution was successful
  and all dependencies are skipped;
- `always`: always build the target regardless of last execution state and results
  of all dependencies (the `.PHONY` target in `make`);
- `artifacts`: a list of files/directory must be present after the execution of
  the target (aka. the output of the target), in relative path to current `.hmake`
  file, or if it's absolute path, it's relative to project root.

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

#### Target Expansions

_Target Expansion_ is useful when defining multiple targets with similar properties.
For example cross compiling for multiple targets, most of the properties are identical
only OS and CPU arch is different. In stead of duplicate the same blocks for each
target, writing one target with expansion syntax in target name, and _hmake_ will help
generate multiple targets using expansion information.

In above example:

```yaml
targets:
    hmake-linux-[arch:amd64,arm,arm64]:
        description: static linked hmake binary for linux-$[arch]
        after:
            - vendor
        watches:
            - '**/**/*.go'
            - build.sh
        cmds:
            - ./build.sh linux $[arch]
        artifacts:
            - bin/linux/$[arch]/hmake
            - bin/hmake-linux-$[arch].tar.gz
            - bin/hmake-linux-$[arch].tar.gz.sha256sum
```

The target `hmake-linux-[arch:amd64,arm,arm64]` will be expanded into three targets:

- hmake-linux-amd64
- hmake-linux-arm
- hmake-linux-arm64

With `$[arch]` substituted accordingly in each expanded targets.

The syntax is simple: `[var-name:val1,val2,...]` to define an expansion variable
with possible values, and in the properties, `$[var-name]` will be substituted.
If `var-name` is not defined in target name, `$[var-name]` will NOT be substituted.
Specially, `$[$]` substitutes to `$`.

Multiple variables can be defined, example:

```
target-[os:linux,darwin]-[arch:386,amd64]
```

will expand to

- `target-linux-386`
- `target-linux-amd64`
- `target-darwin-386`
- `target-darwin-amd64`

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
