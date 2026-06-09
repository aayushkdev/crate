# Entrypoint, User And execve

The last step of startup is to stop being Crate and become the requested program.

This happens in `internal/runtime/init.go`.

## Entrypoint Rules

Crate combines image entrypoint, image command and user command in `internal/container/command.go`.

The rules are:

* entrypoint plus user command when both exist;
* entrypoint plus image command when no user command was provided;
* entrypoint alone when only entrypoint exists;
* user command when no entrypoint exists;
* image command when no user command exists;
* error when there is no command at all.

Crate does not implicitly run a shell. It builds an argument vector and executes it directly.

## PATH Lookup

If the command contains `/`, Crate checks that path directly.

Otherwise, it searches the container `PATH`. If the environment does not define `PATH`, Crate uses:

```text
/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

This happens after the root switch, so paths are resolved inside the container filesystem.

## User Lookup

The container user can come from the image config or from `--user`.

Crate supports:

* `USER`
* `UID`
* `USER:GROUP`
* `UID:GID`

Named users and groups are resolved by reading `/etc/passwd` and `/etc/group` from the container filesystem. Numeric users are used directly, with a best-effort lookup for the primary GID.

Crate applies the result with `setgid` and `setuid`.

## execve

Finally, Crate calls:

```text
execve(path, argv, env)
```

This replaces the internal init process with the container program. The PID does not change. Inside the PID namespace, this process is PID 1.

## No Implicit Shell

Crate does not turn commands into:

```sh
sh -c "..."
```

unless the image entrypoint or user command explicitly asks for a shell. This matters for quoting, environment expansion and signal behavior.

For example:

```sh
crate run alpine echo '$HOME'
```

executes `echo` with the literal argument `$HOME`. A shell would expand it first. Direct execution is simpler and closer to how image entrypoints are represented.

## Entrypoint Examples

Suppose an image has:

```text
Entrypoint: ["/entrypoint.sh"]
Cmd: ["server"]
```

If the user runs:

```sh
crate run image
```

Crate executes:

```text
/entrypoint.sh server
```

If the user runs:

```sh
crate run image shell
```

Crate executes:

```text
/entrypoint.sh shell
```

If the image has no entrypoint and the user runs:

```sh
crate run image sleep 10
```

Crate executes:

```text
sleep 10
```

These rules are simple, but documenting them matters because command resolution is a frequent source of container confusion.

## User Lookup Happens After Root Switch

When Crate resolves a named user, it opens `/etc/passwd`. By the time user application happens, `/etc/passwd` is the container's file, not the host's.

That is the correct behavior. A user named `nginx` inside the image does not have to exist on the host.

Numeric users behave differently. If the user passes `--user 1000`, Crate can use UID 1000 directly. It only tries to find a primary GID as a convenience. If no passwd entry exists for that UID, it falls back to using the UID as the GID.

## Working Directory

Crate applies the configured working directory before applying the user and executing the program.

If the working directory does not exist, Crate warns and falls back to `/`.

This is less strict than some runtimes, but it keeps simple containers from failing only because an image or command requested a missing directory. A future OCI-compatible mode might choose stricter behavior.

## execve As The Point Of No Return

`execve` replaces the process image. After the call succeeds, there is no Crate init code left in that process to recover or perform more setup.

That is why all runtime preparation must happen before the final call:

* rootfs switch;
* standard mounts;
* bind mounts;
* loopback setup;
* network interface wait;
* working directory;
* user selection;
* command resolution.

If any of those steps fails, Crate exits init with an error instead of executing a partially configured container.
