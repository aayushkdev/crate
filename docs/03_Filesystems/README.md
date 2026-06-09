# Filesystems

The filesystem work is where the container's view of the world becomes different from the host's.

Crate starts with a rootfs directory made from image layers. During launch, the internal init process makes that directory become `/`, mounts standard Linux filesystems, and applies user bind mounts.

## Beginner Terms

A filesystem is the tree of directories and files a process can access.

The root filesystem is the tree a process sees under `/`. For a normal host process, `/` is the host root. For a container process, Crate tries to make `/` point at the unpacked image rootfs.

A mount attaches one filesystem or path at another point in the tree. For example, mounting `proc` at `/proc` makes process information appear there.

A mount namespace gives a process its own view of mounts. Without a mount namespace, mounting `/proc` or changing `/` for the container would affect the host view too.

`pivot_root` and `chroot` are two ways to change the root filesystem view. Crate uses `pivot_root` in rootful mode and `chroot` in rootless mode.

A bind mount takes an existing host path and makes it visible somewhere else, often inside the container.

The main implementation files are:

* `internal/fs/rootfs.go`
* `internal/fs/setup.go`
* `internal/fs/proc.go`
* `internal/fs/sys.go`
* `internal/fs/dev.go`
* `internal/fs/run.go`
* `internal/fs/hostfds.go`
* `internal/runtime/mount.go`
* `internal/container/mount.go`

## Chapters

* [Switching Root](01_Switching_Root.md)
* [proc, sys, dev And run](02_Proc_Sys_Dev_And_Run.md)
* [Bind Mounts](03_Bind_Mounts.md)
