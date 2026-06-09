# Layer Extraction And Image Pruning

Layer extraction happens during container creation, but the logic belongs with image storage because it interprets image layer content.

Crate applies each layer from metadata into:

```text
~/.local/share/crate/containers/<id>/rootfs
```

The implementation is `internal/fs/layer.go`.

## Tar Entries

Crate currently handles:

* directories;
* regular files;
* hard links;
* symbolic links;
* overlay-style whiteouts.

Other tar entry types are ignored for now. That means Crate is good enough for many simple images, but it is not yet a complete OCI image unpacker.

## Whiteouts

Whiteouts are how layers delete files.

If a later layer contains:

```text
.wh.foo
```

Crate removes `foo` from the target directory.

If a later layer contains:

```text
.wh..wh..opq
```

Crate treats the directory as opaque and removes existing entries in that directory.

This is one of those details that is easy to miss when reading high-level container explanations. Without whiteouts, the final root filesystem would contain files that the image author expected to be deleted.

## Removing Tags

`crate rmi` removes repo tags, not blindly every blob. If the metadata record still has other tags, Crate rewrites the metadata with the remaining tags.

If no tags remain, it can delete the metadata and then check whether the config and layers are referenced by any other metadata record.

## Pruning

`crate image prune` scans all metadata, builds a set of referenced blob digests, and removes unreferenced blobs.

The rule is conservative:

```text
delete a blob only when no remaining image metadata mentions it
```

This matters because images share layers. Deleting a layer just because one tag was removed would break another image that still uses the same digest.

## Why Layer Order Matters

Layers are not independent directories that get placed beside each other. They are applied in order onto one target tree.

For example:

```text
layer 1: add /etc/example.conf
layer 2: replace /etc/example.conf
layer 3: delete /etc/example.conf
```

The final root filesystem should not contain the file. If Crate applied layers out of order, or ignored whiteouts, the final tree would be wrong.

This is why container creation reads the layer list from metadata and applies it exactly in order.

## Files, Links And Directories

The layer extractor handles the common tar entries needed by many images.

For directories, it creates the directory with the mode from the tar header. For regular files, it creates parent directories, opens the target with truncation, and copies file contents from the tar reader. For symbolic links, it creates the link with the stored link target. For hard links, it creates another directory entry pointing at the same underlying file.

Each of these operations sounds mundane, but together they reconstruct the image rootfs.

The current implementation does not fully preserve ownership or extended attributes. That is an important production gap. Some images depend on ownership, capabilities or special files. A more complete extractor would need to apply that metadata carefully, and rootless extraction would need additional rules for what can be represented safely.

## Whiteouts As Deletions

A tar archive cannot normally say "delete this file from a previous archive". Container layers need that ability, so overlay-style whiteouts encode deletions as special filenames.

Crate's two whiteout cases are:

```text
.wh.name
.wh..wh..opq
```

The first deletes `name` from the same directory. The second makes a directory opaque by deleting existing entries there.

This is one of the places where image format details leak into runtime behavior. Even though whiteouts are stored as tar entries, Crate must treat them as operations on the target filesystem, not as files to create.

## What Happens During container.Create

The image storage and container creation parts meet here:

```text
container.Create
    -> image.ReadMetadata
    -> mkdir containers/<id>/rootfs
    -> for each layer digest:
           storage.BlobPath
           fs.ApplyLayer
```

At this point the container still does not exist as a process. It is only a directory on disk.

This is a useful mental split:

* image pull gets compressed layer blobs;
* container creation expands those blobs into one rootfs;
* runtime launch makes a process see that rootfs as `/`.

## Prune Safety

Pruning has to be conservative because Crate has no central daemon with a live in-memory reference graph. The reference graph is reconstructed from metadata files.

That leads to a simple algorithm:

1. Read all image metadata.
2. Remove dangling metadata.
3. Build a set of referenced config and layer digests.
4. Walk the blob store.
5. Delete blobs not present in the referenced set.

This may leave some data behind if metadata is malformed or if an operation failed partway through. That is acceptable for safety. A prune operation should prefer leaving extra files over deleting data still needed by a valid image.
