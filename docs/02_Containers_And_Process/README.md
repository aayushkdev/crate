# Containers And Process

A container in Crate is two things:

* a directory containing config, state, logs and rootfs;
* a process launched with selected Linux namespace and process attributes.

Creation prepares files. Launch starts the process. Keeping those two stages separate makes the implementation easier to follow.

## Beginner Terms

A process is a running program. When we say Crate starts a container, the final result is still a normal Linux process, just started with a carefully prepared environment.

A namespace changes what a process can see. A PID namespace changes the process tree view. A mount namespace changes the mount table view. A UTS namespace gives a separate hostname. A network namespace gives a separate network stack. A user namespace changes how user and group IDs are mapped.

PID 1 is the first process inside a PID namespace. In a container, the command Crate executes becomes PID 1 from inside the container.

`execve` is the syscall that replaces the current process image with another program. Crate's internal init process eventually calls `execve`, and after that it is no longer Crate code; it is the container command.

Rootless means the user running Crate is not host root. Crate uses a user namespace so the process can look like root inside the container while mapping back to an unprivileged host user.

The main implementation files are:

* `internal/container/create.go`
* `internal/container/config.go`
* `internal/container/state.go`
* `internal/container/name.go`
* `internal/container/command.go`
* `internal/container/user.go`
* `internal/runtime/run.go`
* `internal/runtime/start.go`
* `internal/runtime/launch.go`
* `internal/runtime/init.go`
* `internal/runtime/userns.go`

## The run Path

`crate run alpine sh` is a convenience operation:

```text
runtime.Run
    -> container.Create
    -> runtime.Start
    -> launchContainer
    -> /proc/self/exe init <id>
    -> syscall.Exec
```

If start fails after creation, `runtime.Run` tries to clean up the partially created container.

## Chapters

* [Container Config, State, Names And IDs](01_Config_State_Names_And_IDs.md)
* [Launching Init And Namespaces](02_Launching_Init_And_Namespaces.md)
* [Rootless User Namespaces](03_Rootless_User_Namespaces.md)
* [Entrypoint, User And execve](04_Entrypoint_User_And_Execve.md)
