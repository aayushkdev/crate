# Chapter 2 - Manifests and Platform Selection

Pulling starts with a tag, but a tag still does not tell the runtime which config blob and which layer blobs to fetch. That translation happens through manifests.

## The Problem

A registry tag may point to:

- a single manifest
- a multi-platform index
- a Docker schema 2 manifest list

The runtime needs one concrete answer: which config digest and which ordered layer digests define the image for this machine.

Without manifest resolution, "pull alpine" is ambiguous.

## How Linux Does It

Again, Linux does not implement manifests. The mechanism is still just HTTP and JSON.

You can ask the registry for a tag and inspect the media type:

```sh
repo=library/alpine
tag=latest

token=$(curl -fsSL \
  "https://auth.docker.io/token?service=registry.docker.io&scope=repository:${repo}:pull" \
  | jq -r .token)

curl -i -fsSL \
  -H "Authorization: Bearer $token" \
  -H "Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json" \
  "https://registry-1.docker.io/v2/${repo}/manifests/${tag}"
```

If the `Content-Type` is an index or manifest list, you still have another decision to make: which platform entry to follow.

## How Crate Uses It

Crate makes the accepted manifest types explicit:

```go
// internal/image/registry.go
const manifestAccept = "" +
    "application/vnd.oci.image.manifest.v1+json, " +
    "application/vnd.docker.distribution.manifest.v2+json, " +
    "application/vnd.oci.image.index.v1+json, " +
    "application/vnd.docker.distribution.manifest.list.v2+json"
```

The recursive resolution logic is the important part:

```go
// internal/image/manifest.go
case "application/vnd.oci.image.index.v1+json",
    "application/vnd.docker.distribution.manifest.list.v2+json":

    for _, m := range idx.Manifests {
        if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
            raw, ct, err := fetchManifestByDigest(ref, m.Digest)
            if err != nil {
                return nil, err
            }
            return resolveManifest(ref, raw, ct) // recurse into the chosen platform manifest
        }
    }
```

When Crate reaches a single manifest, it extracts the exact config and layer digests:

```go
// internal/image/manifest.go
return &ImageManifest{
    ManifestDigest: digest,
    ConfigDigest:   sm.Config.Digest,
    Layers:         layers,
    OS:             osName,
    Architecture:   arch,
}, nil
```

That distinction matters. The manifest digest is the immutable identity of the image Crate pulled. The config digest is just one blob inside that image, used later to recover defaults like `Cmd`, `Env`, and `Entrypoint`.

The raw Linux equivalent is still just JSON parsing, but Crate adds one key policy choice: it hardcodes `linux/amd64` today and stores the selected `os` and `architecture` in local metadata.

> Under the Hood
>
> Manifest resolution is where a runtime stops dealing with names and starts dealing with identities.

> ⚠ Watch out
>
> Platform selection is a policy decision, not a parsing detail. If you choose the wrong platform, every later step can still be "correct" and you will still build the wrong root filesystem.

## Connecting the Dots

Chapter 1 got us to the registry. This chapter turns registry responses into a concrete content graph. The next chapter moves one level lower and explains how those layer digests become a final filesystem tree.

## Try It Yourself

Pull an image with a multi-platform index such as `alpine`, then inspect the manifest code in [`internal/image/manifest.go`](../internal/image/manifest.go). Change the selected platform locally only if you understand the consequences: the runtime will try to unpack whatever digests you choose, even if they do not match the host architecture.

## Key Takeaways

- Tags resolve through manifests, not directly to layer blobs.
- A manifest list adds a platform-selection step before any concrete filesystem exists.
- Crate handles both single manifests and indexes with one recursive resolver.
- Manifest digest and config digest are different objects and serve different roles.
- Platform selection is runtime policy, not just transport logic.
