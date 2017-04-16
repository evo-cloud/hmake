---
title: Wrapper Mode
weight: 10
---
_I have a project using Makefile; I don't want to learn HyperMake. But building
in container is very appealing to me._

_hmake_ provides a special _wrapper mode_ for legacy projects.

## A Simple and Quick Start

```sh
echo '#hmake-wrapper gcc:4.9' >HyperMake
```

Now _hmake_ becomes a wrapper of _make_

```sh
hmake target1 target2
```

is equivalent to

```sh
make target1 target2
```

but inside a container created from `gcc:4.9`.

## Specification

In wrapper mode, _HyperMake_ may contain one or more lines.
The first line has specific format:

```
#hmake-wrapper IMAGE [BUILD-FROM] [BUILD-ARG1] [BUILD-ARG2] ...
```

It must start with `#hmake-wrapper`,
and `IMAGE` is required to specify a docker image.

`BUILD-FROM` is optional, when present, _hmake_ will create a `toolchain` target
to build toolchain image for rest of other targets.
It specifies the location of _Dockerfile_.
And the rest of parameters are interpreted as `--build-args` for `docker build`.

If there's only one line, _hmake_ will invoke `make` with command line arguments.
If there're more lines:

- If second line starts with `#!`,
  all lines from second line are copied as a script,
  and _hmake_ invokes this script as `build` target;
- If second line doesn't start with `#!`,
  a line `#!/bin/sh` plus all lines from second line are copied as a script,
  and _hmake_ executes the script.

## Examples

Wraps over _make_

```
#hmake-wrapper gcc:4.9
```

Wraps over a python script

```
#hmake-wrapper python:2.7
#!/usr/bin/env python
print 'Hello'
```

Wraps over a shell script

```
#hmake-wrapper busybox
echo Hello
```
