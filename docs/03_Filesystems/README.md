# Filesystems

The filesystem work is where the container's view of the world becomes different from the host's.

Crate starts with a rootfs directory made from image layers. During launch, the internal init process makes that directory become `/`, mounts standard Linux filesystems, and applies user bind mounts.

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

