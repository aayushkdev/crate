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

Creating the user namespace is only the first half. A real rootless container also needs UID and GID mappings for more than just container root.

With only a one-line map, container UID/GID `0` is usable, but every other ID is unmapped. If an image tries to `chown` a file to UID 101 or switch to a service user, the kernel rejects it because that identity does not exist in the namespace mapping.

You can see that failure directly:

```sh
unshare --user --map-root-user --fork sh -c '
  touch /tmp/crate-idmap-demo
  chown 101:101 /tmp/crate-idmap-demo
'
```

On a single-ID mapping, that fails because only container ID 0 is mapped.

You can watch Linux install those mappings directly:

```sh
rm -f /tmp/crate-userns.pid
unshare --user --fork sh -c 'echo $$ >/tmp/crate-userns.pid; exec sleep 30' &
child="$(cat /tmp/crate-userns.pid)"

echo deny > "/proc/$child/setgroups"
newuidmap "$child" 0 "$(id -u)" 1 1 100000 65536
newgidmap "$child" 0 "$(id -g)" 1 1 100000 65536

cat "/proc/$child/uid_map"
cat "/proc/$child/gid_map"
kill "$child"
```

On a host with delegated IDs configured in `/etc/subuid` and `/etc/subgid`, that produces a map where:

- container UID 0 maps back to your real host UID
- container UID 1 and above map into your delegated subordinate range

That second range is what lets images use service users such as `nginx`, `daemon`, or `www-data` without becoming root on the host.

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
} else {
    cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUTS |
        syscall.CLONE_NEWPID |
        syscall.CLONE_NEWNS
}
```

Namespace creation and UID/GID map installation are separate steps. Crate creates the user namespace here, then installs the actual ID mappings in the next stage.

Crate reads subordinate ID ranges from the host and turns them into namespace mapping triplets:

```go
// internal/runtime/userns.go
subUIDs, err := readSubIDFile("/etc/subuid", current.Username, current.Uid)
if err != nil {
    return nil, err
}
subGIDs, err := readSubIDFile("/etc/subgid", current.Username, current.Uid)
if err != nil {
    return nil, err
}
```

Those host files are the source of truth for what delegated identity ranges the caller is allowed to use.

Crate keeps container root mapped to the caller's real host IDs, then appends the delegated ranges starting at container ID 1:

```go
// internal/runtime/userns.go
ranges := []idMapRange{{
    containerID: 0,
    hostID:      hostRootID,
    size:        1, // root inside -> calling user outside
}}

nextContainerID := 1
for _, subID := range subIDs {
    ranges = append(ranges, idMapRange{ // append delegated range after root
        containerID: nextContainerID,
        hostID:      subID.start,
        size:        subID.count,
    })
    nextContainerID += subID.count
}
```

That layout preserves the familiar container convention that UID 0 is root inside the namespace while still making a larger delegated identity range available after it.

GID mapping has one extra Linux rule: disable `setgroups(2)` first, then write the map.

```go
// internal/runtime/userns.go
func writeSetgroupsDeny(pid int) error {
    path := fmt.Sprintf("/proc/%d/setgroups", pid)
    if err := os.WriteFile(path, []byte("deny"), 0644); err != nil {
        return fmt.Errorf("write %s: %w", path, err)
    }
    return nil
}
```

This is the kernel rule that makes unprivileged GID mapping possible.

Crate deliberately shells out to the standard helper binaries instead of trying to bypass them:

```go
// internal/runtime/userns.go
if err := runIDMapHelper("newuidmap", pid, ranges.uid); err != nil {
    return err
}
if err := runIDMapHelper("newgidmap", pid, ranges.gid); err != nil {
    return err
}
```

Crate is intentionally using the standard Linux helper path here instead of inventing a runtime-specific mapping mechanism.

The child also stays blocked until the parent finishes mapping, then rootless init re-execs itself so setup runs under the final namespace identity:

```go
// internal/runtime/launch.go
if cfg.Rootless {
    if err := configureRootlessUserNS(cmd.Process.Pid); err != nil {
        _ = syncW.Close()
        _ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
        _ = cmd.Process.Kill()
        return err
    }
}
```

```go
// internal/container/container.go
if os.Getenv(initMappedEnv) != "1" {
    Fatal(cratenet.WaitForParent())
    if cfg.Rootless {
        Fatal(reexecMappedInit(containerID, command))
    }
}
```

The order matters. The child is created first because the maps target a real PID, but it cannot safely continue until the parent has installed those maps.

The self-reexec handoff happens in [`cmd/crate/main.go`](../cmd/crate/main.go):

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

> Under the Hood
>
> A rootless process created in a new user namespace exists before the parent has installed `uid_map` and `gid_map`, but that is not yet a usable container identity model. Crate pauses that first process, installs the mappings from the parent, and re-execs init so filesystem setup, `sethostname`, and later user-sensitive syscalls run with the final namespace credentials in place.

> ⚠ Watch out
>
> User namespaces are not just a convenience flag: without delegated IDs in `/etc/subuid` and `/etc/subgid`, many real images will fail on their first `chown` or service-user setup.

## Connecting the Dots

Creation gave Crate a rootfs and config bundle. This chapter turned that bundle into a process with an isolated kernel view and a believable rootless identity model, so when the runtime switches `/` in the next chapter, the process entering that filesystem can behave like a real multi-user Linux environment instead of a single fake root UID.

## Try It Yourself

Inspect your delegated ID ranges, then run an image that actually needs them:

```sh
grep "^$(id -un):" /etc/subuid /etc/subgid
command -v newuidmap newgidmap
crate run -d nginx
crate ps
```

If rootless multi-ID mapping is working, `nginx` should start instead of failing during `chown` setup, and `crate ps` should show the container as running.

## Key Takeaways

- Namespace isolation happens at process creation time, not after the fact.
- Rootless mode depends on real UID/GID mappings, not just `CLONE_NEWUSER`.
- Crate uses self-reexec to separate orchestration from in-container setup.
- Delegated subordinate IDs are what let rootless containers run normal multi-user images.
