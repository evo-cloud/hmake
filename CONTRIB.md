# Guidelines for Contributing to HyperMake

Thanks for using _HyperMake_ and welcome to contribute any features/patches back
to this project!

## Before You Start

- Make sure you understand the features of _HyperMake_
- Look at [Github Issues](https://github.com/evo-cloud/hmake/issues) to see if
  your feature request/issue has already been submitted
- Be familiar with [Go development](http://golang.org) and [Docker](http://www.docker.com)

## Submit an Issue

An issue can be a feature request or a bug. If possible please put on the
corresponding labels `feature`, `bug`, `enhancement` etc. And the maintainers
may alter the labels and put priority labels.

_Features_

Please clearly define the feature with as more details as possible to help others
easily understand the feature. For example, describing the detailed operation
steps, listing the example usage (command line) will be great help.

_Bugs_

In the title, please briefly describe the problem.
In the comment, please follow the structures below:

```
# Problem Description

Detailed problem description

# Environment

Platform: OS, version
Arch: CPU architecture
Version: hmake version

Other information...

# Reproduce Steps

Steps for reproducing the problem

# Attached Content

E.g. Content of your HyperMake file, project directory structure,
scripts when possible.
```

## Submit a Pull Request

- Make sure there's a corresponding issue submitted in
  [Github Issues](https://github.com/evo-cloud/hmake/issues), arbitrary pull
  requests are unlikely to be accepted;
- Make sure your code has been well formatted, vetted/linted and documented;
- Include issue number in your short commit message (first line), like `#15`;
- Tests must be included, depending on the change, End-to-End tests may be required;
- Make sure there's a single commit.

_TIPS_

>To run format check, vet and lint, you can simply use
```
hmake check -v
```

>To fix format, simply use
```
go fmt -w DIR
```

## Dependencies Needed

- Git
- A Github account
- Go 1.6 or above: install from [golang.org](http://golang.org)
- A list of Go dependencies:
  - gvt: `go get github.com/FiloSottile/gvt`
  - ginkgo: `go get github.com/onsi/ginkgo/ginkgo`
  - gomega: `go get github.com/onsi/gomega`
  - hugo: `go get github.com/spf13/hugo`, if you want to generate sites
- For format, vet and lint
  - go tools: `go get golang.org/x/tools/cmd/...`
  - metalinter: `go get -v github.com/alecthomas/gometalinter && gometalinter --install`

## Steps to Get Started

1. Fork `github.com/evo-cloud/hmake` to your own account (assume `dev`)
2. Create a Go development environment, the following steps are recommended for
   most people, especially for those new to Go:

   ```sh
   mkdir -p ~/workspace/go
   cd ~/workspace/go
   export GOPATH=`pwd`
   export PATH="$GOPATH/bin:$PATH"
   go get github.com/FiloSottile/gvt
   go get github.com/onsi/ginkgo/ginkgo
   go get github.com/onsi/gomega
   mkdir -p src/github.com/evo-cloud
   git clone git@github.com:dev/hmake src/github.com/evo-cloud/hmake
   cd src/github.com/evo-cloud/hmake
   gvt restore
   ```

3. Start developing

   ```sh
   cd ~/workspace/go/src/github.com/evo-cloud/hmake
   # and make sure environment variable GOPATH and PATH are properly set as above

   go build ./   # this will build ./hmake executable
   go install ./ # or if you want to install to $GOPATH/bin

   # make sure docker is running
   # if docker-machine is used (not for Linux)
   eval $(docker-machine env MACHINE-NAME)
   docker version # make sure both client and server versions are displayed

   ./hmake -sv # build all by default
   ./hmake check # check format, run lint
   ./hmake test # run tests
   ./hmake e2e # run end-to-end tests
   ./hmake cover # generate coverage

   # alternatively, use go directly
   go test ./test
   # or
   ginkgo ./test
   go test ./test/e2e
   # or
   ginkgo ./test/e2e
   go test -coverprofile=cover.out -coverpkg=./project ./test
   ```
