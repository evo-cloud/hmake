---
title: Command Mode
weight: 7
---

## Overview

Similar to _make_, the command line arguments are interpreted as targets name
for _hmake_.
In addition to that, _hmake_ introduces a special type of targets - commands.
When these targets are invoked, the command line arguments will be passed through
to the target as arguments.

See [File Format]({{< relref "references/fileformat.md#commands" >}}) for details.

```yaml
---
format: hypermake.v0

targets:
  build:
    description: build the source code
    cmds:
      - ./build.sh

commands:
  pack-for:
    description: pack for specified target
    after:
      - build
    cmds:
      - './pack.sh $1'
```

When invoke

```sh
hmake pack-for ubuntu
```

Will execute target `pack-for` and the rest of the command line is passed as
arguments to the target.
From _HyperMake_, it's possible to reference the arguments, like `$1`, in
command targets.

## Restrictions

Command targets are special, with restrictions:

- It can depends on normal targets, but can't be a dependency;
- It must be the first argument on _hmake_ command line.
