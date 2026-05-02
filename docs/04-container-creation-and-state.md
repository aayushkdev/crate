# Chapter 4 - Container Creation and State

After you have image blobs and layer replay logic, you still do not have a container. You have reusable image data. A container is a concrete runtime instance built from that data.

## The Problem

Creation has to answer a different question from pulling:

- which image should this container use?
- where will its root filesystem live?
- what default command and environment will it inherit?
- how will later commands find it again?

If the runtime does not materialize that state explicitly, `start`, `ps`, `logs`, and `stop` all become much harder.
It also becomes hard to know when a container can be removed safely.

## How Linux Does It

Linux does not have a first-class "container object". You build one out of directories, files, mounts, and processes.

A bare-bones equivalent is just filesystem setup plus metadata:

```sh
id=test123
root=/tmp/crate-demo/$id

mkdir -p "$root/rootfs" "$root/logs"
printf '%s\n' '{"id":"test123","status":"created"}' >"$root/state.json"
printf '%s\n' '{"cmd":["/bin/sh"]}' >"$root/config.json"
```

That is not enough to isolate anything yet, but it shows the core idea: a container is "process + durable state on disk", not only a running PID.

## How Crate Uses It

Container creation lives in [`internal/container/create.go`](../internal/container/create.go):

```go
// internal/container/create.go
ref, err := image.ParseReference(imageName)
if err != nil {
    return "", err
}

if !image.MetadataExists(ref) {
    fmt.Printf("Unable to find image '%s:%s' locally\n", ref.Repo, ref.Tag)
    if err := image.Pull(imageName); err != nil {
        return "", err
    }
}
```

Then Crate allocates the rootfs and applies every layer:

```go
// internal/container/create.go
id := generateID()
rootfs := rootfsDir(id)
if err := os.MkdirAll(rootfs, 0755); err != nil {
    return "", err
}

for _, layer := range meta.Layers {
    path, err := image.BlobPath(layer)
    if err != nil {
        return "", err
    }
    if err := fs.ApplyLayer(path, rootfs); err != nil {
        return "", err
    }
}
```

Creation also snapshots runtime defaults from the image config blob:

```go
// internal/container/config.go
cfg := Config{
    ID:         id,
    Image:      primaryRepoTag(meta), // first local tag recorded for this manifest
    Rootless:   os.Geteuid() != 0, // launch policy captured at create time
    Cmd:        imgCfg.Config.Cmd,
    Env:        imgCfg.Config.Env,
    EntryPoint: imgCfg.Config.Entrypoint,
}
```

Finally, Crate writes the first lifecycle record:

```go
// internal/container/create.go
if err := writeState(id, &State{
    ID:        id,
    Image:     familiarRef(ref),
    Status:    StatusCreated,
    LogPath:   LogPath(id),
    CreatedAt: time.Now().UTC(),
}); err != nil {
    return "", err
}
```

> Under the Hood
>
> `create` is where Crate stops talking about images and starts talking about one specific container instance.

Because image metadata is stored per manifest, not per tag file, `create` resolves the requested tag by scanning local manifest records for a matching `repoTags` entry. That keeps one metadata file authoritative even when tags move.

The same state files are what make `crate rm` simple later. Removal does not talk to a daemon. It refreshes `state.json`, refuses to remove running containers, and deletes the container directory only after that check passes.

## Connecting the Dots

Layers gave us a recipe for constructing a root filesystem. Creation turns that recipe into a specific on-disk bundle with config and lifecycle state. The next chapter uses that bundle to launch a process in new namespaces.

## Try It Yourself

Run:

```sh
id=$(go run ./cmd/crate create alpine)
echo "$id"
```

Then inspect the container directory under Crate's data root. Look at `rootfs`, `config.json`, and `state.json` before ever calling `start`.

After stopping the container, try:

```sh
go run ./cmd/crate rm "$id"
```

and confirm that the container directory disappears.

## Key Takeaways

- A container is a concrete on-disk instance, not just an image plus a PID.
- Crate auto-pulls missing images during creation to keep the workflow simple.
- `config.json` captures process defaults; `state.json` captures lifecycle state.
- `crate rm` works because container state is durable and inspectable on disk.
- `create` is the bridge between immutable image content and mutable runtime state.
