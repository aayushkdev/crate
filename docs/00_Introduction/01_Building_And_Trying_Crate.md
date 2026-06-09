# Building And Trying Crate

Crate is a Linux program. It uses Linux namespaces, mounts, pseudo terminals, `/proc`, process signals and user namespace mapping. Building the binary is straightforward, but running every feature depends on the host environment.

## Building

Crate requires Go 1.20 or newer.

From the project root:

```sh
go build -o crate ./cmd/crate
```

Run it directly:

```sh
./crate images
```

or move the binary into a directory in your `PATH`.

## First Attached Container

Try:

```sh
./crate run alpine sh
```

If `alpine` is not local, Crate pulls it first. Then it creates a container directory, applies layers into `rootfs`, starts an internal init process in namespaces, prepares the filesystem and executes `sh`.

## First Detached Container

```sh
./crate run --name demo -d alpine sleep 60
./crate ps
./crate logs demo
./crate stop demo
./crate rm demo
```

This path shows the daemonless model. `ps`, `logs`, `stop` and `rm` do not talk to a background service. They read state files, check PIDs, send signals and remove directories.

## Useful Host Tools

Rootless features may require:

* `newuidmap`
* `newgidmap`
* `/etc/subuid` ranges
* `/etc/subgid` ranges
* `pasta` for private rootless networking

Without `pasta`, Crate can still run rootless containers, but private networking falls back to `none`.

## What To Look For During The First Run

The first run is not only a smoke test. It is a good way to watch the whole runtime path.

After:

```sh
./crate run alpine sh
```

look at:

```sh
find ~/.local/share/crate -maxdepth 3 -type f | sort
```

You should see image metadata, blobs, container config, state and logs. This is the daemonless model in visible form. A daemon-based runtime would keep more of this behind an API. Crate leaves it in files you can inspect.

## Rootful Versus Rootless Test Runs

Running as your normal user exercises the rootless path:

```sh
./crate run alpine id
```

Running with `sudo` exercises the rootful path:

```sh
sudo ./crate run alpine id
```

The visible output may look similar, but the runtime path is different.

Rootless mode uses a user namespace, writes UID/GID maps and uses `chroot` for the root switch. Rootful mode can use `pivot_root` and can mount `/sys` read-only.

This is worth testing early because many container bugs only appear in one mode.

## Testing Networking

Host networking is easiest to try:

```sh
./crate run -n host alpine wget -qO- https://example.com
```

Rootless private networking depends on `pasta`:

```sh
./crate run -n private alpine wget -qO- https://example.com
```

If `pasta` is not installed, Crate should warn and fall back to `none`. That fallback is intentional and is documented in the networking part.

## Testing Logs

Detached containers write logs to disk:

```sh
./crate run --name log-demo -d alpine sh -c 'echo hello; sleep 1; echo done'
./crate logs log-demo
```

Attached containers also mirror output into the log file. This is useful when checking PTY behavior:

```sh
./crate run --name attached-demo alpine sh
./crate logs attached-demo
```

## Cleaning Up

The quickest cleanup path is:

```sh
./crate container prune
./crate image prune
```

Container prune removes stopped containers. Image prune removes image data that no remaining metadata references.

These are separate because a container rootfs is already unpacked and has its own lifetime. Removing image blobs should not remove an existing container rootfs.
