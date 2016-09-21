# HyperMake Wrapper Mode

The _Wrapper Mode_ is the most quick and simplest way to adopt _HyperMake_
with existing projects which already has full build system, like GNU make.

## One line, just works!

```
echo '#hmake-wrapper dockcross/linux-armv7' >HyperMake
hmake
```

The magic word `#hmake-wrapper` in the beginning of `HyperMake` file indicates
the _Wrapper Mode_ of _HyperMake_.
It runs `make` inside container created from image `dockcross/linux-armv7`.

## Wrapper Mode File Format

### First Line

```
#hmake-wrapper IMAGE Dockerfile-FOR-BUILD BUILD-ARGS
```

- `IMAGE`: required, the docker image used to create the container;
- `Dockerfile-FOR-BUILD`: optional, relative path to a folder containing
  `Dockerfile`,  or full path to file if it's not named `Dockerfile`.
  When this is present, `IMAGE` is built locally from the `Dockerfile` before
  run the wrapped build tool;
- `BUILD-ARGS`: optional, space separated `KEY=VALUE` directly passed to
  `--build-args` option of `docker build` command.

E.g.

```
#hmake-wrapper mytoolchain-armhf toolchain/Dockerfile.armhf ARCH=armhf
```

### Second Line

If `HyperMake` only contains the first line, the wrapped build tool is assumed
to be `make`. The following command `hmake` will run `make` inside the container
passing all command line arguments to `make`.

If there are additional lines in the file, the rest of the lines are written to
a script file, and `hmake` run this script file inside the container, passing
all command line arguments to this script file.

By default, the script file is generated with `#!/bin/sh` as first line, and then
filled with the rest lines in `HyperMake`.

E.g.

```
#hmake-wrapper mytoolchain-armhf toolchain/Dockerfile.armhf ARCH=armhf
set -ex
exec ./build.sh "$@"
```

With this, `hmake` will invoke a script inside container like:

```
#!/bin/sh
set -ex
exec ./build.sh "$@"
```

However, we may not always use shell scripts.
It's possible to write in any scripting language when the second line is explicitly
specified:

```
#hmake-wrapper mytoolchain-armhf toolchain/Dockerfile.armhf ARCH=armhf
#!/usr/bin/env python
import sys
print(sys.argv)
```

## Easy! Huh?

To utilize full features of _HyperMake_, the native [File Format](FileFormat.md)
is recommended.
