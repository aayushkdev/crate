# Command Reference

## Images

Pull an image:

```sh
crate pull <image>
```

List local images:

```sh
crate images
```

Remove image tags:

```sh
crate rmi <image>...
```

Prune unreferenced image data:

```sh
crate image prune
```

## Containers

Create a container without starting it:

```sh
crate create [options] <image>
```

Create and start a container:

```sh
crate run [options] <image> [command] [args...]
```

Start an existing container:

```sh
crate start <container> [command] [args...]
```

Run a command in a running container:

```sh
crate exec <container> <command> [args...]
```

List containers:

```sh
crate ps
crate ps -a
```

Read logs:

```sh
crate logs <container>
crate logs -f <container>
```

Stop containers:

```sh
crate stop <container>...
```

Remove stopped containers:

```sh
crate rm <container>...
```

Prune stopped containers:

```sh
crate container prune
```

## Common Options

For `run` and `create`:

* `--name <name>`
* `--user <user>`
* `-e, --env KEY=value`
* `-v, --volume host:container[:ro]`
* `-n, --network host|none|private`
* `-p, --publish HOST:CONTAINER[/tcp|udp]`

For `run`:

* `-d, --detach`
* `--rm`
* `-w, --workdir <path>`
