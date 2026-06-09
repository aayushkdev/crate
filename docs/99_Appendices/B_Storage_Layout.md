# Storage Layout

Crate stores data under:

```text
~/.local/share/crate
```

When run through `sudo`, Crate tries to use `SUDO_USER` so state still lands under the original user's home directory.

The path definitions are in `internal/storage/paths.go`.

## Top Level

```text
~/.local/share/crate/
    blobs/
    images/
    containers/
```

## blobs

```text
blobs/
    sha256/
        <hash>
```

Image config blobs and compressed layer blobs are stored by digest.

## images

Image metadata records are keyed by manifest digest. They include repo tags, config digest, layer digests, platform and size.

## containers

```text
containers/
    <id>/
        config.json
        state.json
        rootfs/
        logs/
            container.log
            network.log
        network.json
```

`config.json` is the desired setup. `state.json` is the observed lifecycle state. `rootfs` is the unpacked filesystem. Logs and network helper state live beside them.

This layout is the main reason Crate can avoid a daemon.

## Example After Pull

After:

```sh
crate pull alpine
```

the store should contain image metadata and blobs:

```text
images/
    sha256:<manifest-digest>
blobs/
    sha256/
        <config-hash>
        <layer-hash>
```

The metadata record points at the config and layer blobs.

## Example After Create

After:

```sh
crate create --name demo alpine
```

the store should contain a container directory:

```text
containers/
    <id>/
        config.json
        state.json
        rootfs/
```

The state should be `created`, because no process has been launched yet.

## Example After Run

After:

```sh
crate run --name demo -d alpine sleep 60
```

the same container directory should also have logs, and state should include a PID and `running` status:

```text
containers/
    <id>/
        logs/
            container.log
        state.json
```

If private networking starts, `network.json` and `network.log` may appear as well.

## Manual Cleanup Warning

It is possible to delete files manually, but doing so can confuse Crate if only part of a container or image record is removed.

Prefer:

```sh
crate rm <container>
crate container prune
crate rmi <image>
crate image prune
```

when testing normal behavior. Manual deletion is best kept for debugging broken local state.
