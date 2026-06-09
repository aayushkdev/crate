# Networking

Crate has a deliberately small networking model:

* `host`
* `none`
* `private`

The current design focuses on rootless containers, where traditional bridge and NAT setup is not available without extra privileges.

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

