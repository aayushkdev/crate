# Network Modes

Network configuration is stored in the container config, but launch may adjust it depending on the runtime environment.

This split lets Crate remember what was requested while still handling host realities like a missing `pasta` binary.

## host

Host networking means the container does not get a separate network namespace.

The process uses the host's network stack. This is the default for rootful containers.

Port publishing does not make sense in host mode because there is no separate namespace boundary to forward across. Crate drops published ports for modes that cannot use them and returns a warning.

## none

`none` networking creates a network namespace but does not connect it to the outside world.

Crate still brings up loopback. This is done with socket and ioctl calls in `internal/net/loopback.go`:

* create an `AF_INET` datagram socket;
* read interface flags with `SIOCGIFFLAGS`;
* add `IFF_UP`;
* write flags with `SIOCSIFFLAGS`.

This gives programs a working `localhost` even when there is no external network.

## private

`private` networking currently means rootless private networking through `pasta`.

Crate validates that private networking is used only in rootless mode and with the supported backend. A rootful request for private networking fails rather than silently doing something else.

Rootless containers default to private mode. If `pasta` is unavailable at launch, Crate falls back to `none`.

## Defaults

Crate chooses different defaults for rootful and rootless containers:

```text
rootful  -> host
rootless -> private
```

The rootful default is simple because the host already has a network stack and Crate does not yet implement rootful bridge networking.

The rootless default is more interesting. A rootless user generally expects some isolation from the host, but cannot easily set up a bridge and NAT. `pasta` gives Crate a practical private-networking path without a daemon.

## Network Namespace Is Conditional

Crate does not always create `CLONE_NEWNET`.

For `host`, it should not. The entire point of host mode is to share the host network namespace.

For `none` and `private`, it should. Both modes need a separate network namespace:

* `none` leaves it disconnected except for loopback;
* `private` connects it through `pasta`.

This is a useful example of how a high-level flag changes low-level process creation.

## Publishing Ports Only Applies To private

Port publishing is meaningful when there is a boundary between host and container networking.

With host networking, the process already binds host ports directly. With none networking, there is no outside connection to forward. With private networking, Crate can ask `pasta` to forward selected host ports into the namespace.

This is why Crate drops published ports for unsupported modes and warns.

## Resolver Files

Network setup is not only interfaces and port forwarding. Programs also expect resolver files.

Crate writes basic `/etc/resolv.conf` and `/etc/hosts` for private networking. Without these, a container could have an interface but still fail common name-resolution paths.

This is a small but important runtime detail. "Networking works" usually means more than "an interface exists".

## How Mode Affects Launch

The network mode affects launch before any packet is sent.

`host` mode keeps launch simpler:

```text
no CLONE_NEWNET
no loopback setup
no pasta helper
no port publishing
```

`none` mode adds:

```text
CLONE_NEWNET
bring up loopback
```

`private` mode adds:

```text
CLONE_NEWNET
parent starts pasta
child waits for interface
bring up loopback
```

That is the kind of task-to-operation mapping these notes are trying to preserve.

## Why none Is Useful

`none` is not only a fallback. It is useful when testing isolation.

If a program should not reach the network, `none` gives it a separate namespace with only loopback. Programs can still use localhost internally, but they cannot make normal outbound connections through Crate.

This is a simple security and debugging mode.

## What Bridge Networking Would Add

Docker's common rootful networking model uses a bridge. Crate does not implement that yet.

A bridge implementation would need:

* bridge device creation;
* veth pair creation;
* moving one end into the container namespace;
* IP address assignment;
* routes;
* NAT or port forwarding rules;
* DNS between containers.

That is a large subsystem. Keeping it out for now makes the current rootless `pasta` path easier to understand.
