# The `shell` Execution Driver

This is simplest driver which inteprets `script` or `cmds` as shell script/commands.
The following properties are supported:

- `env`: a list of environment variables (the form `NAME=VALUE`) to be used for
  execution; E.g.

  ```yaml
  targets:
      sample:
          env:
              - ARCH=x86_64
              - OS=linux
              - RELEASE        # without =VALUE, the value is populated from
                               # current environment of hmake
  ```

- `script`: a multi-line string represents a full script to execute for the target
  E.g.

  ```yaml
  targets:
      sample:
          script: |
              #!/bin/bash
              echo 'This is a bash script'
      sample1:
          script: |
              #!/usr/bin/env perl
              print "a perl script"
  ```

- `cmds`: when `script` is not specified, this is a list of commands to execute
  for the target E.g.

  ```yaml
  targets:
      sample:
          cmds:
              - mkdir -p bin
              - gcc -o bin/hello hello.c
  ```

- `console`: when `true`, the current stdin/stdout/stderr is directly passed to
  command which is able to fully control the current console, however no output
  can be captured and logged in this case.

The list of `cmds` will be merged as a shell script.
And the intepreter is `/bin/sh`.
`set -e` is inserted as the first line to fail-fast.

## Limits

The commands are always interpreted as shell commands (`sh` or `bash`), on
Windows, _shell driver_ doesn't work.
