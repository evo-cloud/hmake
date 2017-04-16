---
title: FAQ
weight: 11
---
- **Q**: Why I can't `git clone` private repositories?<br>
  **A**: HyperMake runs the build inside containers which may not have the right
  SSH keys or credentials. There're two options:

  - Mapping `~/.ssh` into the container using `volumes` property:

    ```yaml
    volumes:
      - '~/.ssh:/src/.ssh:ro'
    ```
  - Mapping `~/.netrc` into the container:

    first, create a file `~/.netrc`. See the
    [manual](https://www.gnu.org/software/inetutils/manual/html_node/The-_002enetrc-file.html)
    for the format and content. E.g.

    ```
    machine github.com
    protocol https
    login username
    password password
    ```

    Then use `volumes` property to map into the container.

    ```yaml
    volumes:
      - '~/.netrc:/src/.netrc:ro'
    ```

- **Q**: Can I build projects for Windows?<br>
  **A**: Depending on toolchain. HyperMake builds on Linux, for C/C++,
 install [mingw](http://www.mingw.org) toolchain in the container and do
 the cross complication.

- **Q**: Can I build projects for native Mac OS?<br>
  **A**: Depending. If it's a project in Go, yes. If it depends on native Mac OS
  libraries, it's possible when cross compiling toolchain and libraries are
  installed on Linux.

- **Q**: Why target is skipped but output file is not present?<br>
  **A**: Property `artifacts` is not specified in the target. _HyperMake_ checks
  both input and output files to determine if a target is up-to-date. Property
  `watches` lists the input files whose last modification time is checked, and
  property `artifacts` lists the output files whose presence is checked.
  If `artifacts` is not specified, _HyperMake_ assumes the target doesn't generate
  output files.

- **Q**: What's the `artifacts` if the target doesn't output files?<br>
  **A**: No need to specify `artifacts` if there's no output file. Some targets
  like docker build doesn't output files, it will automatically check if the
  image exists. To explicitly rebuild the target, use `-r TARGET`, `-b` or `-R`
  options.

- **Q**: I want to run some commands, which are specific to my local environment,
  before certain targets. But I don't want to put them in `HyperMake` file.<br>
  **A**: You can create a `.hmakerc` in project root, and exclude that file using
  `.gitignore`. The `settings` in `.hmakerc` will override those in `HyperMake`
  and use `before` to inject your local targets into `HyperMake`, e.g.

  ```yaml
  ---
  format: hypermake.v0
  targets:
    pre-build:
      description: my local task before build
      before:
        - build
      cmds:
        - do something
  settings:
    property: my-value
  ```

- **Q**: How to map a volume from a folder relative to project root?<br>
  **A**: In top-level `HyperMake`, use relative path for source of the volume,
  in `*.hmake` files under sub-directories, prefix `-/` to a relative path. E.g.

  ```yaml
  targets:
    example:
      volumes:
        - '-/run:/var/run'
  ```

  Anyway, in `volumes`, prefix `-/` can always be used to indicate a path
  relative to project root. Please read [Docker Driver](DockerDriver.md) for
  details.

- **Q**: Where can I find the output of my target after running `hmake`?<br>
  **A**: `hmake` creates a hidden folder `.hmake` under project root. The output
  of a target is saved in `.hmake/TARGET.log`.

- **Q**: Does `hmake` print logs?<br>
  **A**: Yes. `hmake` writes its own debug logs in `.hmake/hmake.debug.log`.

- **Q**: What're the recommended entries in `.gitignore`?<br>
  **A**: Put the following entries in `.gitignore`:

  ```
  .hmake
  .hmakerc
  ```
