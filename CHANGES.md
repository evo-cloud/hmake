# Change Log

## v1.3.1

_Release Date_: 2017-04-28

- Add port mapping (`-p`) support in docker driver
- Rewrite documents and site pages

## v1.3.0

_Release Date_: 2017-02-08

- Introduce _background task_ to support `docker-compose`
- Introduce _command mode_
- Pass arguments in wrapper mode
- Fix dependency handling due to unclear success mark
- Strip symbols and debug info

## v1.2.0

_Release Date_: 2016-10-06

- Target expansion
- Retire example _drone_

## v1.1.1

_Release Date_: 2016-09-20

- Wrapper mode
- Upgrade dependency [clix.go](https://github.com/codingbrain/clix.go)

## v1.1.0

_Release Date_: 2016-08-31

- Patching `/etc/passwd` with valid username for certain apps (like `git`) to work;
- Extract `~` as `$HOME` in host path of volume mapping;
- Direct `docker commit` support in target, via `commit` property;
- Unify implementation logic with/without `docker-machine`;
- Properties and values in target are also calculated in target digest;
- `cmd` and `script` are calculated in target digest;
- Target name validation;
- Options `-s` and `-v` are default;
- In-container execution support (`-x` and `-X` options);
- Direct `docker push` support in target, via `push` property;

## v1.0.0

_Release Date_: 2016-06-26

Initial stable release
