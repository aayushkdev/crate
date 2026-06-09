# Stop, Remove, Prune And Future Work

Stopping and removing containers is mostly state validation plus signals and filesystem cleanup.

## stop

`crate stop` reads refreshed state first. If the container is not running, there is nothing to stop.

For a running container, Crate:

1. Marks it as `stopping`.
2. Sends `SIGTERM` to the process group.
3. Waits for the process to exit.
4. Sends `SIGKILL` if it does not exit in time.
5. Tears down network helper state.
6. Marks the container stopped.
7. Removes it if `--rm` was set.

Signals are sent to the process group so detached child processes owned by the container are more likely to be terminated together.

## rm

`crate rm` removes stopped containers. It refuses to remove running containers.

The remove operation deletes the container directory:

```text
containers/<id>
```

Because config, state, rootfs and logs live under that directory, this removes the container's durable state.

## container prune

`crate container prune` scans containers and removes stopped ones.

This is the container equivalent of image pruning: scan filesystem state, decide what is safe, remove only that.

## What Docker Still Has

Crate does not yet implement many Docker lifecycle features:

* `exec` into running containers;
* `inspect`;
* resource stats;
* restart policies;
* health checks;
* pause and unpause;
* event streams;
* cgroups;
* seccomp;
* capabilities;
* named volumes;
* bridge networking;
* Dockerfile builds.

A good future Crate feature would be a trace mode that logs higher-level internal steps:

```text
image: resolve manifest
create: apply layer
launch: clone namespaces
rootless: write uid/gid maps
fs: pivot_root or chroot
net: start pasta
exec: execve /bin/sh
```

That would match the purpose of the project: not just using the abstraction, but seeing the internal steps that make it work.

## Why Stop Uses Process Groups

Detached containers may start child processes. If Crate only signalled the top-level PID, children could continue running after the main process receives a signal.

Crate starts detached containers in a separate process group and sends signals to the process group by using a negative PID. This is not a perfect init system, but it is better than signalling only one process.

A more complete runtime would also consider subreapers, init shims and cgroups for stronger cleanup.

## Auto Remove

`--rm` records an auto-remove flag in the container config.

When the container exits or is stopped, Crate checks that flag and removes the container directory. This is why auto-remove belongs in config rather than state: it is a user-selected policy for that container.

The cleanup path still has to be careful. If launch fails after creation, `runtime.Run` tries to remove the container. If removal fails because the process started, it tries to stop and then remove.

## Image Prune Versus Container Prune

Crate has two prune concepts:

* image prune removes unreferenced image data;
* container prune removes stopped containers.

They are intentionally separate. Removing a stopped container should not remove the image it came from. Removing an unreferenced image blob should not remove a container rootfs that has already been unpacked.

The stores are related, but their lifetimes are different.

## Future: exec

`exec` into a running container would be one of the most educational next lifecycle features.

It would require entering the namespaces of an existing process, usually through `/proc/<pid>/ns/*`, then starting a new process with the container's filesystem, environment and user rules.

This would teach another important Linux mechanism:

```text
setns
```

It would also force Crate to decide how much of the original container config should apply to later exec sessions.

## Future: cgroups

Cgroups are one of the biggest missing Docker features.

Namespaces answer:

```text
what can the process see?
```

Cgroups answer:

```text
how much can the process use?
```

CPU, memory, pids and IO limits all belong here. Cgroups would also improve cleanup because Crate could track and kill all processes in a container's cgroup rather than relying only on process groups.

## Future: Trace Mode

A trace mode should probably come before full syscall tracing.

The first useful version could emit structured internal events:

```text
pull.resolve_reference
pull.fetch_manifest
create.apply_layer
launch.clone
rootless.write_uid_map
fs.pivot_root
net.start_pasta
exec.execve
```

This would directly support the documentation style. Readers could run a command and see the same task breakdown the notes explain.
