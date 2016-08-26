# HyperMake Command Line

## Usage

```
hmake [OPTIONS] [TARGETS]
```

There's no specific order between `OPTIONS` and `TARGETS`. All `OPTIONS` starts
with hyphen `-` while `TARGETS` doesn't.

## Option Parsing Rules

Common Unix command line option parsing rule is adopted:

- A short option starts with a single hyphen and then a single letter
  (e.g. `-C`);
  it may have a corresponding long option which starts with a double hyphen
  followed by a word (or a few words concated with hyphen)
  (e.g. `--chdir`, `--this-is-a-multi-word-opt`);
- The value after a short option is provided as a separated argument
  (after a space, e.g. `-C /tmp/proj`);
  for long option, the value follows directly with `=` in the same argument
  (e.g. `--chdir=/tmp/proj`);
- Some option can be specified multiple times to provide a list or a map
  (e.g.<br>
  _list_: `--include=a.hmake --include=b.hmake` or `-I a.hmake -I b.hmake`;<br>
  _map_: `--define=ARCH=x86_64 --define=OS=linux` or `-D ARCH=x86_64 -D OS=linux`
  );
- Bool options can be specified without value as `true` or prefixed by `no-` as
  `false` (e.g. `--verbose` for `true`, `--no-verbose` for `false`);
  It can also be specified with a value in the long option format
  (e.g. `--verbose=true` or `--verbose=false`)

## Options

- `--chdir=PATH, -C PATH`: Chdir to specified PATH first before doing anything
- `--file=FILE, -f FILE`: Override the default project file name `HyperMake`
- `--include=FILE, -I FILE`: Include additional files (must be relative path under project root), can be specified multiple times
- `--property=key=value, -P key=value`: Define property in global `settings` section, `key` may include `.` to specify the hierarchy
- `--parallel=N, -p N`: Set maximum number of targets executed in parallel, 0 for auto, -1 for unlimited
- `--rebuild-all, -R`: Force rebuild all needed targets
- `--rebuild-target TARGET, -r TARGET`: Force rebuild specified target, this can repeat
- `--rebuild, -b`: Force rebuild targets specified on command line
- `--skip TARGET, -S TARGET`: Skip specified target (mark as Skipped), this can repeat
- `--exec, -x`: Execute a shell command in the context of a target.
  The target name must be specified in `settings.exec-target` or use `--exec-with=TARGET`.
  It's extremely useful to run arbitrary command in the context of a target.
  It should come as the last option, as the rest command-line arguments will be
  treated as command.

  For example:
  ```sh
  hmake -x go version
  hmake -x   # enter an interactive shell inside the container
  ```

  The commands parsing after `-x` is directly executed by `execvp` system call,
  not a command to be parsed by shell. So shell syntax like `&&` won't work.

  To run as a shell command
  ```sh
  hmake -x /bin/sh -c 'go version || echo "go version failed"'
  ```

- `--exec-with=TARGET, -X TARGET`: Explicitly specify the target for `--exec` instead of
  fetching from `settings.exec-target`.
  As it implies `--exec`, it should come as the last option.

  For example:
  ```sh
  hmake --exec-with=vendor go version
  ```

- `--json`: Dump execution events to stdout each encoded in single line json
- `--summary, -s`: Show execution summary before exit
- `--quiet, -q`: Suppress output from each target
- `--rcfile|--no-rcfile`: Load .hmakerc inside project directories, default is true
- `--color|--no-color`: Explicitly specify print with color/no-color
- `--emoji|--no-emoji`: Explicitly specify print with emoji/no-emoji
- `--no-debug-log`: Disable writing debug log to `hmake.debug.log` in hmake state directory
- `--show-summary`: When specified, print previous execution summary and exit
- `--targets`: When specified, print list of target names and exit
- `--dryrun`: When specified, run targets as normal but without invoking execution drivers (simply mark task Success)
- `--version`: When specified, print version and exit

The parsing of options stops when `--` is encountered. The rest of arguments will be
treated as target names. Except `--exec`/`--exec-with` already implies end of
options parsing, `--` should not be used.

> Note: `--exec`/`--exec-with` doesn't affect the last execution result of the
> target, though it displays the result and updates the summary. So the target
> may still be skipped next time if nothing changed.

## Exit Code

- 0: Success
- 1: One or more targets failed
- 2: Incorrect usage
