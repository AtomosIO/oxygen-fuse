oxygen-fuse
===========
Atomos Oxygen FUSE Filesystem

This package mounts the Atomos Oxygen file system to a local directory on your computer. It uses the FUSE interface supplied with most modern Linux based kernels to avoid needing changes to the kernel or new drivers.

By mounting the remote file system to a local directory, oxygenfuse allows you to more easily manipulate files using tools available on your local computer such as IDEs and scripts.


### Mount Atomos Oxygen to a local directory:
```sh
cd oxygenfuse
go get
go build
mkdir <Mount Location>
oxygenfuse http://oxygen-dot-atomos-release.appspot.com <Mount Location> <Token>
```

A Token can be obtained by POSTing to http://oxygen-dot-atomos-release.appspot.com/tokens/ with the following JSON packet:
```js
{
  "user": "<Username or Email>",
  "password": "<Password>",
}
```
The response of the request above will include a new Token to be used for in the Token field.
