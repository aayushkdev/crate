# Task To Syscall Map

This appendix maps Crate tasks to the low-level Linux operations involved.

It is not a full syscall trace. Go itself performs many syscalls that are not specific to containers. This map focuses on the calls that explain the container runtime tasks.

## Launch Internal Init

Task:

```text
start /proc/self/exe init <id> in selected namespaces
```

Important operations:

* `clone` through Go's `exec.Cmd`;
* `CLONE_NEWUTS`;
* `CLONE_NEWPID`;
* `CLONE_NEWNS`;
* `CLONE_NEWUSER` in rootless mode;
* `CLONE_NEWNET` for isolated network modes;
* `setpgid` for detached containers.

Reference:

* `internal/runtime/launch.go`

## Rootless User Mapping

Task:

```text
make container root map to the current host user
```

Important operations:

* write `/proc/<pid>/setgroups`;
* execute `newuidmap`;
* execute `newgidmap`;
* re-execute internal init after mapping.

Reference:

* `internal/runtime/userns.go`
* `internal/runtime/init.go`

## Root Filesystem Switch

Task:

```text
make containers/<id>/rootfs become /
```

Important operations:

* `mount("", "/", "", MS_PRIVATE|MS_REC, "")`;
* bind mount rootfs onto itself;
* `pivot_root` in rootful mode;
* detached unmount of old root;
* `chroot` in rootless mode;
* `chdir("/")`.

Reference:

* `internal/fs/rootfs.go`

## Standard Container Mounts

Task:

```text
provide /proc, /sys, /dev and /run
```

Important operations:

* mount `proc`;
* mount read-only `sysfs` in rootful mode;
* mount `tmpfs` on `/dev`;
* bind mount or create device nodes;
* mount `devpts`;
* mount `tmpfs` on `/run`.

Reference:

* `internal/fs/setup.go`
* `internal/fs/proc.go`
* `internal/fs/sys.go`
* `internal/fs/dev.go`
* `internal/fs/run.go`

## Bind Mounts

Task:

```text
mount a host path into the container
```

Important operations:

* open source before root switch;
* prepare destination path;
* `mount(..., MS_BIND|MS_REC, ...)`;
* remount with `MS_RDONLY` when requested.

Reference:

* `internal/container/mount.go`
* `internal/runtime/mount.go`

## Enter The Container Program

Task:

```text
replace Crate init with the requested command
```

Important operations:

* `sethostname`;
* `setgid`;
* `setuid`;
* `execve`.

Reference:

* `internal/runtime/init.go`
* `internal/container/user.go`
* `internal/container/command.go`

## Bring Up Loopback

Task:

```text
make localhost usable in an isolated network namespace
```

Important operations:

* create an `AF_INET` datagram socket;
* `ioctl(SIOCGIFFLAGS)`;
* set `IFF_UP`;
* `ioctl(SIOCSIFFLAGS)`.

Reference:

* `internal/net/loopback.go`

## Stop A Container

Task:

```text
terminate a running container process group
```

Important operations:

* liveness check with signal `0`;
* send `SIGTERM` to process group;
* wait;
* send `SIGKILL` if needed;
* send `SIGTERM` to `pasta` helper during network teardown.

Reference:

* `internal/runtime/stop.go`
* `internal/runtime/process.go`
* `internal/net/pasta.go`

