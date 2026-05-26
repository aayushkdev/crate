# Chapter 10 - Rootless Networking with pasta

Once a container has its own PID, mount, and user namespaces, it can still be using the host network stack. That is often convenient, but it is not a private container network.

## The Problem

Without a network namespace, a container shares the host's interfaces, routes, and loopback device. That means there is no separate `lo`, no container-specific interface to configure, and no clean place to attach a rootless network helper.

Adding `CLONE_NEWNET` closes that gap, but it creates a new one immediately. A fresh network namespace starts almost empty. Even `localhost` is unusable until the loopback interface is brought up, and outbound networking does not exist until something connects that namespace to the outside world.

For a rootless runtime, the real before/after is:

- before: the process can run, but it either shares the host network or has no usable network at all
- after: the process gets a private network namespace, `lo` works, and a userspace helper can connect that namespace without making the runtime itself rootful

## How Linux Does It

Linux gives you the network namespace. It does not give an unprivileged bridge, NAT setup, or a magical rootless `veth` pair that just works.

That distinction matters. Rootful container networking is usually built from kernel objects such as:

- `veth` pairs
- bridge devices
- routing changes
- packet forwarding and NAT rules

Those are a good fit when the runtime already has host-level privilege. They are not a good fit for a fully rootless runtime because the runtime would need to mutate shared host networking state.

`pasta` solves a different problem. It is a userspace helper that joins an existing network namespace and forwards traffic on behalf of the process inside it. Crate does not ask the kernel for a rootful bridge topology. It asks Linux for a private network namespace, then runs an ordinary userspace process that makes that namespace useful.

You can observe the blank starting point first:

```sh
unshare --user --map-root-user --net --fork sh -c '
  echo "before:"
  ip link show lo
  ip link set lo up
  echo "after:"
  ip addr show lo
'
```

That shows the first rootless networking truth: a fresh network namespace is empty enough that even loopback is down.

Then you can hand that namespace to `pasta`:

```sh
unshare --user --map-root-user --net --fork sh -c 'echo $$; sleep 30' &
child=$!
pasta --config-net --ns-ifname crate0 "$child"
```

What `pasta` is doing here is not creating a bridge on the host. It is joining the target namespace, creating the namespace-local interface, and proxying connectivity in userspace.

That is why it works well for rootless operation:

- no host bridge device is required
- no host-side `veth` plumbing is required
- no host-wide firewall/NAT programming is required
- the helper can target one existing process by PID and exit when that process disappears

That is also why Crate does not treat `pasta` as the rootful networking design even though root could run it. Once the runtime is rootful, a kernel-native `veth` plus bridge design is usually the better long-term model because it gives the runtime direct control of topology, routing, and port publishing instead of outsourcing those jobs to a userspace forwarder.

## How Crate Uses It

Crate treats networking as a policy decision before launch, not as something the child figures out after it starts.

```go
// internal/net/config.go
func DefaultConfig(rootless bool) Config {
    if rootless {
        return Config{
            Mode:          ModePrivate,
            Backend:       "pasta", // rootless private network by default
            InterfaceName: DefaultInterfaceName,
        }
    }
}
```

This is where Crate decides whether the container shares the host stack, gets a private namespace with connectivity, or gets a private namespace with no external networking at all.

Users can override that policy on `crate run` and `crate create` with `--network` or `-n`:

```sh
crate run -n host alpine ip addr
crate run -n none alpine ip addr
crate run -n private alpine ip addr
```

The accepted modes are:

- `host`: share the host network stack
- `none`: create a network namespace, bring up loopback, and provide no external connectivity
- `private`: create a network namespace and use the configured backend, currently `pasta` for rootless containers

Port publishing is meaningful only in `private` mode:

```sh
crate run -n private -p 8080:80 nginx
```

If `-p` is supplied with `host` or `none`, Crate warns, drops the publish entries from the stored config, and continues. In `host` mode the process is already using host ports directly; in `none` mode there is no external network path to publish through.

Crate resolves that policy at runtime and degrades cleanly when `pasta` is unavailable:

```go
// internal/net/runtime.go
if _, err := exec.LookPath("pasta"); err != nil {
    fallback := cfg
    fallback.Mode = ModeNone
    fallback.Backend = ""
    fallback.InterfaceName = ""
    fallback.Publish = nil
    return fallback, "pasta not installed; using none networking", nil
}
```

That fallback is deliberate. The container keeps its private namespace and loses connectivity instead of silently regaining host-level network sharing. If published ports were configured, the warning also explains that the mappings are ignored because the effective mode is `none`.

Launch-time namespace selection happens in the parent:

```go
// internal/runtime/launch.go
if cratenet.RequiresNetNS(netCfg) {
    cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNET // create a private netns only when needed
}
```

This is the point where Crate stops sharing the host network stack and asks Linux for a separate network view.

Crate blocks the child until parent-side setup is finished:

```go
// internal/runtime/launch.go
if cfg.Rootless || cratenet.RequiresNetNS(netCfg) {
    cmd.ExtraFiles = append(cmd.ExtraFiles, syncR) // child blocks on this pipe
    cmd.Env = append(cmd.Env, cratenet.SyncEnv(3)) // pass the read end into init
}
```

```go
// internal/net/sync.go
_, err = file.Read(buf) // parent writes one byte only after setup succeeds
```

The parent/child pipe is the launch barrier. The child cannot run ahead into a half-configured namespace.

When the helper is needed, Crate starts `pasta` against the container process PID:

```go
// internal/net/pasta.go
cmd := exec.Command(
    "pasta",
    "--config-net",
    "--ns-ifname", cfg.InterfaceName,
    strconv.Itoa(pid), // target the container's namespace by PID
)
```

This is the handoff from namespace creation to userspace networking. `pasta` targets the container process PID and configures the namespace-local interface from outside the container.

Crate also writes the network-side state it needs for teardown:

```go
// internal/net/state.go
type State struct {
    Backend       string `json:"backend"`
    HelperPID     int    `json:"helper_pid"`
    InterfaceName string `json:"interface_name"`
}
```

That state file is just enough bookkeeping to stop the helper cleanly later and explain what networking mode the container actually got.

The child does the namespace-local finishing work. First, it reuses the parent's mode decision:

```go
// internal/container/container.go
cfg.Network = cratenet.ApplyModeOverride(cfg.Network)
```

The child is not recalculating network policy. It is consuming the parent's already-resolved decision.

Then it brings up loopback inside isolated namespaces:

```go
// internal/container/container.go
if cfg.Network.Mode == cratenet.ModeNone || cfg.Network.Mode == cratenet.ModePrivate {
    Fatal(cratenet.BringUpLoopback())
}
```

```go
// internal/net/loopback.go
req.Flags |= unix.IFF_UP
_, _, errno := unix.Syscall(..., uintptr(unix.SIOCSIFFLAGS), ...) // mark lo up
```

This is what turns `localhost` back into a working interface inside the isolated namespace.

For private mode, the child also waits for the helper-created interface to appear before continuing:

```go
// internal/container/container.go
if cfg.Network.Mode == cratenet.ModePrivate {
    Fatal(cratenet.WaitForInterface(cfg.Network.InterfaceName, 5*time.Second))
}
```

The child waits because interface creation is asynchronous from its point of view. The namespace exists before the helper has necessarily finished wiring it.

Crate also copies resolver and hosts data into the container rootfs before the helper runs:

```go
// internal/net/files.go
func writeResolvConf(rootfs string) error {
    data, err := os.ReadFile("/etc/resolv.conf") // inherit host resolver config
    if err != nil {
        return err
    }

    path := filepath.Join(rootfs, "etc", "resolv.conf")
    return os.WriteFile(path, data, 0644) // seed the container view before exec
}
```

Resolver and hosts files are part of making the namespace believable to ordinary userspace, not an optional extra.

Teardown is symmetric: if `pasta` was started, Crate records its PID and later terminates it during stop or exit handling.

```go
// internal/net/pasta.go
if state.HelperPID > 0 {
    _ = syscall.Kill(state.HelperPID, syscall.SIGTERM)
}
```

Teardown is equally explicit. If the helper was started, the runtime owns stopping it.

> Under the Hood
>
> `pasta` is not replacing network namespaces. It is filling in the piece a rootless runtime does not want to own at the host level: forwarding and interface setup for one specific container namespace. That is why Crate still needs `CLONE_NEWNET`, loopback setup, resolver files, and synchronization. `pasta` is the transport side of the design, not the whole design.

> ⚠ Watch out
>
> A fresh network namespace with loopback still down is more broken than it looks: even software that never leaves the machine often assumes `127.0.0.1` works.

## Connecting the Dots

This chapter turns "a process with isolated kernel and filesystem views" into "a process with its own network view" without making the runtime rootful. At that point Crate has covered the core container surfaces most users notice immediately: filesystem, process tree, identity, terminal behavior, lifecycle, and now network reachability.

## Try It Yourself

Run a detached container and inspect the runtime state Crate wrote for networking:

```sh
id=$(crate run -d alpine sleep 30)
cat "$HOME/.local/share/crate/containers/$id/state.json"
test -f "$HOME/.local/share/crate/containers/$id/network.json" && \
  cat "$HOME/.local/share/crate/containers/$id/network.json"
crate stop "$id"
```

If `pasta` is installed, `network.json` should exist and include the helper PID plus the `crate0` interface name. If `pasta` is not installed, `state.json` should still show `"network_mode": "none"` after Crate prints the warning about using none networking.

## Key Takeaways

- `CLONE_NEWNET` gives isolation, but not a usable network by itself.
- Rootless private networking in Crate is implemented with a userspace helper, not a rootful bridge or veth setup.
- Loopback, resolver files, and child/parent synchronization are part of making a network namespace usable, not optional polish.
- Falling back to `none` preserves isolation when the helper is unavailable instead of silently widening access through host networking.
