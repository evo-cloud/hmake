# Questions and Best Practices

- **Q**: Why I can't `git clone` private repositories?
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
