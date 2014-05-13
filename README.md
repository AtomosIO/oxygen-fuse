How to start developing oxygen-fuse-fs:
  - mkdir ~/atomos
  - cd ~/atomos
  - git clone https://github.com/hesamrabeti/oxygen-fuse-fs
  - git clone https://github.com/githubnemo/CompileDaemon.git
  - cd CompileDaemon
  - go build
  - cd ../oxygen-fuse-fs/oxygenfs
  - ../../CompileDaemon/CompileDaemon -command="go test -cover" &



<<<<<<<<<<<<<<<<<< TEXT BELOW IS FROM BAZIL.ORG/FUSE <<<<<<<<<<<<<<<<<<

bazil.org/fuse -- Filesystems in Go
===================================

`bazil.org/fuse` is a Go library for writing FUSE userspace
filesystems.

It is a from-scratch implementation of the kernel-userspace
communication protocol, and does not use the C library from the
project called FUSE. `bazil.org/fuse` embraces Go fully for safety and
ease of programming.

Here’s how to get going:

    go get bazil.org/fuse

Website: http://bazil.org/fuse/

Github repository: https://github.com/bazillion/fuse

API docs: http://godoc.org/bazil.org/fuse

Our thanks to Russ Cox for his fuse library, which this project is
based on.

------------------------------------
