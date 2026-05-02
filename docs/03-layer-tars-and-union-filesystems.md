# Chapter 3 - Layer Tars and Union Filesystems

An image is not one monolithic root filesystem archive. It is a stack of filesystem deltas.

## The Problem

Container images are built from layers because layers make image distribution and reuse practical:

- unchanged base layers can be shared
- small changes do not require shipping a whole root filesystem again
- build systems can cache intermediate steps

The runtime problem is that a layer is not a full filesystem. It is a change-set that must be applied in order, including deletions.

## How Linux Does It

Linux itself does not define OCI layer whiteouts, but it does provide the filesystem primitives used to replay them: create files, create links, remove paths, and mount union-capable filesystems such as overlayfs.

You can observe whiteout-like behavior manually with tar archives:

```sh
mkdir -p /tmp/layers/base/etc /tmp/layers/upper/etc
printf 'one\n' >/tmp/layers/base/etc/example
printf 'two\n' >/tmp/layers/upper/etc/example

tar -C /tmp/layers/base -czf /tmp/base.tar.gz .
tar -C /tmp/layers/upper -czf /tmp/upper.tar.gz .
```

A real OCI layer also needs a way to express deletion. Overlay-based image formats do that with whiteout entries such as `.wh.filename` and `.wh..wh..opq`.

## How Crate Uses It

Crate does not mount overlayfs. It teaches the model by replaying layer tarballs into a plain directory.

The replay loop is in [`internal/fs/layer.go`](/home/aayush/projects/crate/internal/fs/layer.go):

```go
// internal/fs/layer.go
for {
    hdr, err := tr.Next()
    if err == io.EOF {
        break
    }

    target := filepath.Join(rootfs, hdr.Name)
    base := filepath.Base(hdr.Name)

    if strings.HasPrefix(base, ".wh.") {
        if base == ".wh..wh..opq" {
            dir := filepath.Dir(target)
            entries, _ := os.ReadDir(dir)
            for _, e := range entries {
                _ = os.RemoveAll(filepath.Join(dir, e.Name())) // make directory opaque
            }
        } else {
            orig := strings.TrimPrefix(base, ".wh.")
            _ = os.RemoveAll(filepath.Join(filepath.Dir(target), orig)) // delete lower-layer path
        }
        continue
    }
```

Then Crate materializes the surviving tar entries:

```go
// internal/fs/layer.go
case tar.TypeReg:
    f, err := os.OpenFile(
        target,
        os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
        os.FileMode(hdr.Mode),
    )
    if err != nil {
        return err
    }
    if _, err := io.Copy(f, tr); err != nil {
        f.Close()
        return err
    }
```

The equivalent raw Linux idea is "apply ordered filesystem mutations", but Crate makes the replay engine concrete instead of hiding it behind overlayfs.

> Under the Hood
>
> Overlayfs composes lower and upper directories at mount time. Crate composes them at unpack time. Same model, different tradeoff.

> ⚠ Watch out
>
> Layer order is semantic, not cosmetic. Reordering layers changes the final filesystem.

## Connecting the Dots

The manifest chapter produced an ordered list of layer digests. This chapter explains what those digests mean. The next chapter uses that replay engine to build a per-container root filesystem and write the first runtime state to disk.

## Try It Yourself

Pull an image, find one unpacked container later under Crate's data directory, and inspect how files that were added, overwritten, or deleted across layers appear in the final rootfs. Then read the whiteout handling code in [`internal/fs/layer.go`](/home/aayush/projects/crate/internal/fs/layer.go) with that result in mind.

## Key Takeaways

- Image layers are ordered filesystem deltas, not complete filesystems.
- Deletions are encoded explicitly through whiteout entries.
- Crate chooses unpack-time composition over mount-time composition for clarity.
- Union filesystem ideas still apply even when overlayfs itself is not used.
