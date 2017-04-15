ugo
-------

ugo is an extremly simple and obvious solution to all your GOPATH troubles.

If [the official way of working with go](https://golang.org/doc/code.html#Workspaces) is somehow troubling you,
this is the one workaround you need.



| tested with         | works  |
|-------------------  |------- |
| go1.7.4 linux/amd64 | yep    |

please open a github issue to report if it works on your platform (windows is not supported)


the fix
-----
simply add a file ".gopackage" to your package root, containing the full path of  your package


```sh
$ git clone github.com/foo/bar && cd something
$ go get
  error: you have angered the GOPATH god
$ echo 'github.com/foo/bar' > .gopackage
$ u go get
$ u go build
# just bloody works
```

- A workaround for the underlying design mistake in go, not a vendoring tool (see background)
- So trivial that many other tools can be implemented on this
- Foward and backward compatible with the official way of working.
- A Trivial single file with zero overhead to the project that all projects can adopt painfree

using ugo
-----

This repo contains ugo, that creates a workspace in the project root of whatever you're building.

It's quite simplistic and many more tools can be built using the obvious package fix (read below).
If you implement something else that uses .gopackage, let me know and i'll link from here


```sh
git clone https://github.com/aep/ugo.git
ln -s $PWD/ugo/ugo ~/bin/u
```

"u anything" will create the virtual workspace, add it to your existing gopath and then exec "anything"

this also works with other tools need to know the import path,
such as generators or parsers, for example to use ginkgo generate just prefix it with "u"

```sh
$ ginkgo generate
Couldn't identify package import path.
$ u ginkgo generate
#works fine
```


You can also pass -r to this script to not add but replace the workspace, which works exactly the same way,
except that all dependencies are redownloaded for every project. I'd recommend using vendoring instead.
-r also serves as a workaround for broken tools that only accept a single path in GOPATH


background
------

GOPATH has many issues, such as crippling any workflow with private repos, mixed go and non-go code, git submodules, or global dependencies generally being a terrible idea.
Probably a million more.
There's workarounds for every issue GOPATH brings, but the puzzling part for me was: why does GOPATH even exist, when no other language has it?

Here's an attempt to explain the situation:

let's assume this is your project:

```sh
  $ git clone github.com/company/server/
  $ tree server
  server
  ├── router
  │   └──  router.go
  └── cmd
      └──  main.go
```

your router.go:

```go
package router
```

and your main.go imports the router sub package

```go
package main
import "github.com/company/server/router"
```


since package names must be short, and import names must be full urls
there is no way go build could figure out these two things are related.
Unless we're complying with [the law](https://golang.org/doc/code.html#Workspaces) :

```sh
$ cd $GOPATH
$ git clone github.com/company/server/ src/github.com/company/server/
$ tree
  src
  └── github.com
      └── company
          └── server
              ├── router
              |   └── router.go
              └── cmd
                  └──  main.go

```

now the go tools understand what "import github.com/company/server/router" means,
because that is a valid path relative to GOPATH

I'm unsure wether this is the reason gopath exists, but it is the only time it's actually needed for any of the go tools.

ugo fakes that layout inside a symlink, so the go tools know where to find stuff,
without you having to bother about all the other problems that come with gopath

```
  $ git clone github.com/company/server/
  $ cd server && ugo *anything*
  $ tree server
  server
  ├── .gopackage
  ├── router
  │   └──  router.go
  ├── cmd
  │   └──  main.go
  └── .workspace
      └── src
          └── github.com
              └── company
                  └── ...

```

In theory, go tools could just read .gopackage directly or use full names in the package directive,
and do away with GOPATH. Until then, this workaround hopefully enable a saner workflow for everyone.
