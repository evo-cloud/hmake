---
title: Exec(Debug) Mode
weight: 6
---

**exec mode** allows execution of arbitrary command inside the container under
the context of the specified target.
It's very useful to debug the environment.

## Overview

```yaml
---
format: hypermake.v0

targets:
  build:
    description: build the source code
    cmds:
      - ./build.sh
    env:
      - FLAG=1

  test:
    description: run test
    cmds:
      - ./test.sh

settings:
  exec-target: build
  docker:
    image: 'ubuntu:latest'
```

Sometimes, developers need to get into the execution environment of a target to
find out what goes wrong.
_exec mode_ is designed for this specific needs.

With above _HyperMake_, run

```
hmake -x
```

It will bring up a shell `/bin/sh` inside the container under the context of
`build` target.

The default shell is `/bin/sh`, to override it, specify `exec-shell` in `settings`:

```yaml
---
settings:
  exec-shell: /bin/bash
```

The shell is brought up in interactive mode, that means _hmake_ will run in
interactive mode, printing build progress of all dependencies, and run the
target in a TTY (`docker run -it`).

An arbitrary command can be specified instead of invoking an interactive shell.
In this case, _hmake_ works in non-interactive mode, only the stdout/stderr of
the target will be printed on the console.

```
hmake -x sh -c 'echo $FLAG'
```

It will print `1`.

{{% notice info %}}
If command is specified, it's a exec command line, not interpreted by shell
{{% /notice %}}

## Alternative Target Context

By default, `-x` will use target context specified by `exec-target`.
If a different target context is needed, use `-X target`:

```sh
hmake -X test
hmake -X test command arg1 arg2 ...
```

{{% notice info %}}
Option `-x` and `-X` indicates the end of option parsing from the command line,
so they must be the last option, and `--` should not be used after that.
If used, `--` will be directly interpreted as common/argument to execute.
{{% /notice %}}

## Run as root

Usually target specified by `exec-target` doesn't run as root.
However it's useful to run `hmake -x` as root user, for example inspecting
installed packages and checkout new packages etc.

Use settings property to override this

```
hmake -P docker.user=root -x
```
