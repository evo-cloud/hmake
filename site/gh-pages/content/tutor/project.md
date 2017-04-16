---
title: A HyperMake Project
weight: 1
---
## Project Layout

A HyperMake project requires one `HyperMake` file in root directory of the
project (The file name can be overridden using `--file|-f`,
see [Command Line]({{< relref "references/commandline.md" >}}) for details).
The command `hmake` can be invoked in any sub-directory of the project and it will
look up `HyperMake` to determine the root of project.

Similar to a _Makefile_, _HyperMake_ defines targets with dependencies
(see [File Format]({{< relref "references/fileformat.md" >}})).
To better organize the targets, it's possible to break targets into multiple files.
Files other than the root _HyperMake_ are modules and named with suffix
`.hmake`.
Use `includes` section to include all modules into one _HyperMake_ file.


```text
ProjectRoot
  |
  +--module1/
  | +--module.hmake
  +--module2/
  | +--module.hmake
  +--HyperMake
  +--.gitignore
```

In _HyperMake_

```
---
format: hypermake.v0
name: sample

includes:
  - **/**/*.hmake
```

`name` is required in root _HyperMake_, but not in `*.hmake` modules.

In `.gitignore`, the following two lines are recommended:

```
.hmake
.hmakerc
```

## State Directory

When `hmake` runs, it will create a state folder `.hmake` in the project root directory.
Within the folder, it tracks the execution result of targets, hmake logs,
output of targets, checksum of mtime of watched files and artifacts, and
also other files including generated scripts.

In most cases, this directory is irrelevant to the project and can be safely ignored.
However, it's very useful for analyzing the scripts/logs if something goes wrong.

## Local Modules

`hmake` respects special files named `.hmakerc` when they are present.
They are the same as modules (`*.hmake`), but loaded after them.
The files specified after `--include|-I` are treated the same way but loaded last.

Here's the order to load files:

- Load _HyperMake_ in project root;
- Load all modules from `includes` section, recursively;
- Find and load `.hmakerc` files from the folder launching `hmake` up to project root;
- Load files specified by `--include|-I`

Usually, `.hmakerc` and files specified by `--include|-I` are excluded from
being committed to source control (in `.gitignore`), and are called _local modules_.
These files are used to override settings and
[inject dependencies]({{< relref "#dependency-injection" >}}) for
the local needs of a developer.

{{% notice info %}}
Local modules can override settings, inject dependencies, defining additional targets.
Existing targets can't be overridden/redefined.
{{% /notice %}}

## Targets

Targets are the basic execution units of a HyperMake project.
They are defined in `targets` section:

```
---
format: hypermake.v0

targets:
  build:
    description: build the source code
    cmds:
      - ./build.sh
```

Targets have dependencies, which are defined using `before` and `after`:

```
---
format: hypermake.v0

targets:
  toolchain:
    description: build docker image contains toolchain
    build: toolchain
  build:
    description: build the source code
    after:
      - toolchain
    cmds:
      - ./build.sh
settings:
  docker:
    image: mytoolchain
```

When run `hmake build`, `toolchain` will execute before `build`.

`before` means the opposite, and is mostly used for dependency injection.

## Dependency Injection

This usually happens in `.hmakerc` and files specified by `--include|-I`.
In the file, define a target with `before` property, e.g. in `.hmakerc`

```
---
format: hypermake.v0

targets:
  pre-build:
    description: hook before build
    after: toolchain
    before: build
    cmds:
      - patch some files
```
