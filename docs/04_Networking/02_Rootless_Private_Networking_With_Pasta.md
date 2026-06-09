# Rootless Private Networking With pasta

Rootless users cannot normally create host bridges, veth pairs and iptables NAT rules. Crate uses `pasta` to provide private networking without requiring a rootful daemon.

The implementation is in `internal/net/pasta.go`.

## Launch Sequence

Private networking needs the container child PID. The parent starts the child, performs any required user namespace setup, then starts `pasta` before releasing the child from the sync pipe.

The helper command includes:

```text
--config-net
--ns-ifname crate0
<pid>
```

The container waits for the interface to appear before continuing.

## Port Publishing

Crate accepts:

```text
HOST:CONTAINER[/tcp|udp]
```

It validates:

* host port range;
* container port range;
* protocol;
* duplicate host ports per protocol.

Published ports become `pasta` TCP and UDP specs. If only TCP ports are published, UDP is set to `none`, and the other way around.

## Network Files

Before starting `pasta`, Crate writes basic files into the rootfs:

* `/etc/resolv.conf`
* `/etc/hosts`

This gives common programs a resolver and hostname baseline inside the container.

## State And Teardown

Crate writes network helper state to:

```text
containers/<id>/network.json
```

and logs helper output to:

```text
containers/<id>/logs/network.log
```

On teardown, Crate reads `network.json`, sends `SIGTERM` to the helper PID, and removes the state file.

## Fallback

If `pasta` is missing, `ResolveRuntimeConfig` changes the runtime mode to `none`, clears backend/interface/published port fields, and prints a warning.

This keeps rootless containers usable even on systems without the helper installed.

## Why pasta Fits Rootless Crate

Traditional container networking often uses:

* a bridge device;
* veth pairs;
* routing rules;
* NAT rules;
* firewall changes.

Those are usually rootful operations. A daemon can perform them, but Crate deliberately avoids a daemon and supports ordinary users.

`pasta` is useful because it can attach networking to a process namespace without Crate implementing a full bridge driver. This keeps networking understandable while still making rootless containers practical.

## Parent-Side Helper

The `pasta` helper is started by the parent, not by the final container program.

The parent knows the host PID of the child. It can start `pasta`, write network helper state, and then release the child through the sync pipe. The child can then wait for the expected interface name before continuing.

This division prevents the container command from needing to know about the helper at all.

## Interface Name

Crate uses:

```text
crate0
```

as the default interface name inside the namespace.

The child waits for that interface in private mode. If the interface never appears, startup fails rather than executing a program with half-configured networking.

## TCP And UDP Specs

Crate stores published ports as structured values:

```text
host port
container port
protocol
```

When building `pasta` arguments, it separates TCP and UDP specs. This matters because `pasta` expects them as different options.

If the user publishes only TCP, Crate explicitly tells `pasta` not to forward UDP. That avoids accidental broad forwarding.

## Teardown Is State-Based

Because Crate has no daemon, teardown cannot rely on a live in-memory object for the helper.

Instead, setup writes:

```text
network.json
```

with the helper PID. Stop and cleanup paths read that file later. If the file is missing, teardown treats networking as already gone.

This mirrors the rest of Crate's daemonless lifecycle model.

## Failure Modes

Private networking can fail in several practical ways:

* `pasta` is not installed;
* the helper starts and exits quickly;
* the expected interface never appears;
* a port publish specification is invalid;
* a host port is already in use;
* resolver files cannot be written into the rootfs.

Crate handles the missing-binary case before launch by falling back to `none`. Other failures usually abort startup, because running with half-configured private networking would be misleading.

## Why Fallback Uses none

The fallback could have been host networking, but that would weaken isolation silently.

If the user requested rootless private networking and Crate cannot provide it, `none` is a safer fallback than sharing the host network. The program may lose network access, but it does not unexpectedly gain host-network access.

This is a design choice worth preserving: fallbacks should not silently remove isolation.

## Future Rootful Networking

A future rootful network implementation would likely be separate from `pasta`.

It might create a bridge, veth pair and NAT rules. That would teach a different set of Linux networking concepts:

* network links;
* moving interfaces between namespaces;
* assigning addresses;
* route setup;
* packet forwarding;
* firewall/NAT rules.

Keeping that out of the current implementation makes the rootless path easier to follow.
