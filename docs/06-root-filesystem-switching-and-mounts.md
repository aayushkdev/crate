# Chapter 6 - Root Filesystem Switching and Special Mounts

After the child process enters new namespaces, it still starts with the host's root filesystem unless the runtime actively changes that.

## The Problem

If the process can still see the host root, the image you unpacked does not matter much. The runtime must switch `/` to the container rootfs and then mount a few special filesystems that normal userspace expects:

- `/proc`
- `/sys`
- `/dev`
- `/run`

This chapter is where the unpacked directory tree becomes a believable operating environment.

## How Linux Does It

Two classic mechanisms matter here:

- `chroot(2)`: changes path resolution root, but does not fully replace the mount tree
- `pivot_root(2)`: swaps the current root with a new mount and detaches the old one

You can experiment with `chroot` quickly:

```sh
sudo mkdir -p /tmp/rootfs/bin
sudo cp /bin/sh /tmp/rootfs/bin/
sudo chroot /tmp/rootfs /bin/sh
```

`pivot_root` is stricter and more representative of container runtimes because it changes the active mount root:

```c
// pivot_root_demo.c
#define _GNU_SOURCE
#include <sys/mount.h>
#include <sys/syscall.h>
#include <unistd.h>

int main(void) {
    syscall(SYS_pivot_root, "/newroot", "/newroot/.oldroot");
    chdir("/");
    umount2("/.oldroot", MNT_DETACH);
    return 0;
}
```

Mounting `/proc` is also just an ordinary mount syscall:

```sh
mount -t proc proc /proc
```

## How Crate Uses It

Rootfs switching starts in [`internal/fs/rootfs.go`](/home/aayush/projects/crate/internal/fs/rootfs.go):

```go
// internal/fs/rootfs.go
if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
    return err // stop mount propagation first
}
if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
    return err // ensure rootfs is a mount point before pivot_root
}
if rootless {
    if err := setupChroot(rootfs); err != nil {
        return err
    }
} else {
    if err := setupPivotRoot(rootfs); err != nil {
        return err
    }
}
```

The `pivot_root` path is explicit:

```go
// internal/fs/rootfs.go
putold := filepath.Join(rootfs, ".oldroot")
if err := os.MkdirAll(putold, 0700); err != nil {
    return err
}

if err := syscall.PivotRoot(rootfs, putold); err != nil {
    return err
}

if err := syscall.Unmount("/.oldroot", syscall.MNT_DETACH); err != nil {
    return err
}
```

After the root switch, Crate mounts the support filesystems:

```go
// internal/fs/setup.go
if err := mountProc(); err != nil {
    return err
}
if err := mountSys(rootless); err != nil {
    return err
}
if err := mountDev(rootless, hostFDs); err != nil {
    return err
}
if err := mountRun(); err != nil {
    return err
}
```

The `/dev` setup is worth reading carefully because rootless and rootful paths differ materially in [`internal/fs/dev.go`](/home/aayush/projects/crate/internal/fs/dev.go).

> Under the Hood
>
> `pivot_root` is about mount topology, not just pathname filtering. That is why it is stronger than `chroot`.

> ⚠ Watch out
>
> If you forget to make mounts private before switching roots, mount propagation can leak behavior between host and container in unpleasant ways.

## Connecting the Dots

Namespaces isolated the kernel view. This chapter isolates the filesystem view and reconstructs the minimum pseudo-filesystems userspace expects. The next chapter chooses and `exec`s the actual workload that will become PID 1.

## Try It Yourself

Read [`internal/fs/rootfs.go`](/home/aayush/projects/crate/internal/fs/rootfs.go) and compare the `rootless` and `rootful` paths. Then start a container and inspect whether `/proc`, `/sys`, `/dev`, and `/run` exist as expected inside it.

## Key Takeaways

- A container rootfs is not active until the runtime switches `/` to it.
- `pivot_root` changes the mount root; `chroot` only changes path resolution.
- `/proc`, `/sys`, `/dev`, and `/run` are reconstructed explicitly by Crate.
- Rootless launch changes the rootfs strategy because some mount operations are more constrained.
