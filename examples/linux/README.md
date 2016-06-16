# Example - Build Linux Kernel

This example builds linux kernel for multiple platforms using HyperMake.

## Get Started

```sh
hmake -C examples/linux -sv
```

And collect outputs from `build/out/PLATFORM/arch/ARCH/boot`

## Structure

On top directory, `HyperMake` defines the project and top-level targets.
And some scripts implements the build related logic which can be easily called
in other hmake targets.

The `builders` directory contains `*.hmake` files which define targets building
docker images with required toolchain.

The `targets` directory contains sub-directories for different platforms.
Each sub-directory contains a `config` file which is used as kernel config file,
and a `.hmake` file defining targets to build/clean the kernel.

When build starts, an intermediate directory `build` is created.
`build/src` contains the Linux kernel source, and `build/out/PLATFORM` is created
for output of specific platform.
By building the kernel in separated platform directories, it's possible to build
kernel for multiple platform in parallel.

## Add a new platform

It's very easy to add a new platform:

- Create folder `targets/PLATFORM`
- Generate/Copy `config` (kernel config) to `targets/PLATFORM`
- Create `targets/PLATFORM/target.hmake` containing hmake targets of:
  - `target-PLATFORM`: it builds the kernel
  - `clean-PLATFORM`: it removes `build/out/PLATFORM`

The recommended naming convention for `PLATFORM` is `ARCH-BOARD`,
e.g. `arm-vexpress` is to build kernel for VExpress board with ARM CPU.

## Other targets

In `HyperMake`, additional targets are defined to help build the kernel:

- `nconfig`/`menuconfig`: these maps to `make nconfig/menuconfig`. It helps you
  to edit the kernel config file. The config file is saved in `build/out/config/.config`.
  After finishing the config, you can copy this file to your platform folder.
  These targets also demonstrate the use of `console` property in hmake to allow
  interactive targets.

## Build with a different kernel version

The kernel version is hard-coded in `fetch.sh`.
To use a different kernel version, simply update `fetch.sh`.
