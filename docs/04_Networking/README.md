# Networking

Crate has a deliberately small networking model:

* `host`
* `none`
* `private`

The current design focuses on rootless containers, where traditional bridge and NAT setup is not available without extra privileges.

## Beginner Terms

A network namespace gives a process a separate view of network interfaces, routes and ports.

Host networking means the container shares the host network namespace. There is no separate container network.

None networking means the container has its own network namespace but no external connection. Crate still brings up loopback so `localhost` works.

Private networking means the container gets a separate network namespace that is connected through a helper. Crate currently uses `pasta` for this in rootless mode.

Port publishing means forwarding a host port to a container port. This only makes sense when the container has a separate network namespace.

Loopback is the local network interface used for `localhost` and `127.0.0.1`.

The main implementation files are:

* `internal/net/config.go`
* `internal/net/runtime.go`
* `internal/net/pasta.go`
* `internal/net/ports.go`
* `internal/net/files.go`
* `internal/net/loopback.go`
* `internal/net/state.go`
* `internal/net/sync.go`

## Chapters

* [Network Modes](01_Network_Modes.md)
* [Rootless Private Networking With pasta](02_Rootless_Private_Networking_With_Pasta.md)
