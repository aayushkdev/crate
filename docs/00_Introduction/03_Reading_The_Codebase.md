# Reading The Codebase

Crate's packages are divided by runtime responsibility.

| Path | Responsibility |
|---|---|
| `cmd/crate` | Cobra CLI commands and flag parsing. |
| `internal/image` | Image references, registry requests, manifests, blobs, metadata and image pruning. |
| `internal/container` | Container config, state, names, users, commands, mounts and removal. |
| `internal/runtime` | Run/start/stop/logs/ps, namespace launch, internal init, PTYs and process finalisation. |
| `internal/fs` | Layer extraction and filesystem setup. |
| `internal/net` | Network modes, `pasta`, loopback, resolver files and network state. |
| `internal/storage` | Shared paths and JSON read/write helpers. |

## Main run Path

For:

```sh
crate run alpine sh
```

the source path is:

```text
cmd/crate/run.go
    -> internal/runtime/run.go
        -> internal/container/create.go
            -> internal/image
            -> internal/fs.ApplyLayer
        -> internal/runtime/start.go
        -> internal/runtime/launch.go
            -> /proc/self/exe init <id>
                -> internal/runtime/init.go
                    -> internal/fs.Setup
                    -> internal/runtime.applyMounts
                    -> internal/net
                    -> internal/container.ResolveEntrypoint
                    -> internal/container.ResolvePath
                    -> syscall.Exec
```

Creation and launch are separate. Creation prepares files. Launch starts a process.

## Main storage Path

The storage path starts at:

```text
internal/storage/paths.go
```

Most commands eventually use paths from there. The JSON helper in `internal/storage/store.go` writes through a temporary file and renames it into place.

## Main rootless Path

Rootless launch crosses:

```text
internal/runtime/launch.go
internal/runtime/userns.go
internal/runtime/init.go
internal/fs/rootfs.go
internal/net/runtime.go
internal/net/pasta.go
```

That is where the differences from rootful mode are easiest to see.

## Suggested Reading Order

Start with the command you care about.

For `pull`, read:

```text
cmd/crate/pull.go
internal/image/reference.go
internal/image/registry.go
internal/image/manifest.go
internal/image/pull.go
internal/image/metadata.go
internal/image/store.go
```

For `run`, read:

```text
cmd/crate/run.go
internal/runtime/run.go
internal/container/create.go
internal/runtime/start.go
internal/runtime/launch.go
internal/runtime/init.go
```

For filesystem setup, read:

```text
internal/fs/setup.go
internal/fs/rootfs.go
internal/fs/proc.go
internal/fs/sys.go
internal/fs/dev.go
internal/fs/run.go
internal/runtime/mount.go
```

For lifecycle commands, read:

```text
internal/container/state.go
internal/runtime/ps.go
internal/runtime/logs.go
internal/runtime/stop.go
internal/container/remove.go
internal/container/prune.go
```

## Reading Patterns

Crate often follows a pattern:

```text
parse user input
normalise into internal config
validate early
write durable state
perform Linux operation
update state
```

For example, port publishing starts as CLI strings, becomes structured `PublishedPort` values, is validated, stored in container config, translated into `pasta` arguments at launch, and displayed by `ps` later.

This kind of flow is worth tracking because the same concept appears at several levels of abstraction.

## Where To Look For Linux Boundaries

The strongest Linux boundaries are in:

* `syscall.SysProcAttr` clone flags;
* `syscall.Mount`;
* `syscall.PivotRoot`;
* `syscall.Chroot`;
* `syscall.Setuid` and `syscall.Setgid`;
* `syscall.Exec`;
* `unix.Syscall` for network ioctls.

When a chapter says "namespace", "mount" or "exec", look for one of those calls. They are the places where Crate stops being ordinary data manipulation and asks the kernel to change process state.
