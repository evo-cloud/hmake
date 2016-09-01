# How HyperMake Helps Make Life Easier

## On-board New Team Member

Most software projects have complicated requirements to setup a development
environment. There are always a list of software to install, including a
specific version of toolchain, specific versions of libraries, etc.

The new member on-board experience often suffers, and usually takes days up to
weeks.

With _HyperMake_, the on-board experience is simplest. The new member only need
to install [docker](https://www.docker.com) and `hmake` executable.
Then the project will be built smoothly.

## Deliver an Open Source Project

_Compiling from source_ is challenging work for most of people.
However, it's unavoidable if new features are required and a pre-built package is
not available.
If the project is built using _HyperMake_, it makes _compiling from source_ the
simplest thing on the world.

## Build/Test Product Consistently

People often complain _environment issue_ when building/testing a software product.
Because the _environments_ (toolchain, libraries) can't always be identical from
developer to developer. Things work fine in one environment will likely be
broken in another environment.
With help of _HyperMake_, the product is always built/tested in a clean and consistent
way because the same docker image is used across all different environments.

## Platform-independent Development

With _HyperMake_, it no longer requires developers work on specific platform.
As long as the project can be built on Linux, the developer is free to choose working
on Linux/Mac OS/Windows.
