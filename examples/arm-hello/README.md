# Hello World built for ARM

This example demonstrates the use of _HyperMake Wrapper Mode_ to quickly build
a project which requires a special toolchain. It just works!

## Hello World

The project is a very simple _Hello World_ C program, however, we want to build
it for ARM processors.
The cross compiler `arm-linux-gnueabihf-gcc` is required, but not installed on
the host.

Installing a cross compiler is not easy, though we can use `apt-get ...`, it won't
make host system clean.

By using docker image `dockcross/linux-armv7` which always contains the toolchain,
the wrapper `HyperMake` file simply makes it built!
