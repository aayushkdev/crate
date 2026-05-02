# Chapter 5 - Namespaces and Rootless Launch

This is where Crate becomes a runtime rather than an image unpacker. A root filesystem alone is just a directory tree. A container needs a process with an isolated kernel view.

## The Problem

Without namespaces, a process started from the container rootfs still shares the host's:

- PID space
- hostname
- mount table
- user identity model

That means no meaningful isolation. The runtime needs a way to create a child process whose world view differs from the host.

## How Linux Does It

Linux creates namespace boundaries at process creation time.

You can observe this directly with `unshare`:

```sh
sudo unshare --uts --pid --mount --fork sh -c '
  hostname crate-demo
  echo "inside pid: $$"
  hostname
  ps -o pid,comm
'
```

For rootless behavior, user namespaces are the important extra piece:

```c
// clone_userns.c
#define _GNU_SOURCE
#include <sched.h>
#include <stdio.h>
#include <unistd.h>

int main(void) {
    printf("host uid=%d gid=%d\n", getuid(), getgid());
    return 0;
}
```

The key kernel interface is `clone(2)` or equivalent process creation with flags such as `CLONE_NEWUSER`, `CLONE_NEWPID`, `CLONE_NEWUTS`, and `CLONE_NEWNS`.

## How Crate Uses It

Crate re-execs itself and attaches namespace flags through `SysProcAttr`.

```go
// internal/runtime/launch.go
cmd := exec.Command("/proc/self/exe", args...)
cmd.SysProcAttr = &syscall.SysProcAttr{}
if cfg.Rootless {
    cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUSER |
        syscall.CLONE_NEWUTS |
        syscall.CLONE_NEWPID |
        syscall.CLONE_NEWNS

    cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
        {ContainerID: 0, HostID: os.Getuid(), Size: 1}, // map container root to caller uid
    }
    cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
        {ContainerID: 0, HostID: os.Getgid(), Size: 1},
    }
    cmd.SysProcAttr.GidMappingsEnableSetgroups = false
} else {
    cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUTS |
        syscall.CLONE_NEWPID |
        syscall.CLONE_NEWNS
}
```

The self-reexec handoff happens in [`cmd/crate/main.go`](/home/aayush/projects/crate/cmd/crate/main.go):

```go
// cmd/crate/main.go
if len(os.Args) >= 3 && os.Args[1] == "init" {
    containerID := os.Args[2]
    command := os.Args[3:]

    container.InitContainer(containerID, command)
    return
}
```

That split matters:

- parent process chooses policy and records lifecycle state
- child process enters namespaces and becomes container init logic

> Under the Hood
>
> Crate does not ask Linux to "make a container". It asks Linux to make a process with new namespaces, then builds container semantics around that process.

> ⚠ Watch out
>
> User namespaces are not just a convenience flag. They change what "root" means inside the container and what privileged operations remain possible.

## Connecting the Dots

Creation gave us a rootfs and config bundle. Namespaces give the soon-to-be-container process its isolated kernel view. The next chapter changes not just what the process sees in the kernel, but what it sees as `/`.

## Try It Yourself

Create a container, then compare running it rootless versus rootful if your environment allows both. Observe hostname, PID numbering, and mount behavior from inside the container process.

## Key Takeaways

- Namespace isolation happens at process creation time, not after the fact.
- Rootless mode depends on user namespace mappings, not just dropping privileges.
- Crate uses self-reexec to separate orchestration from in-container setup.
- A rootfs without namespaces is still just a directory tree on the host.
