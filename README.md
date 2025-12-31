# crate

Crate is a small implementation of containerisation, inspired by how Docker works internally.


### Run

```bash
sudo go run ./cmd/crate run <ROOTFS> <COMMAND> [ARG...]
```

Example
```bash
sudo go run ./cmd/crate run rootfs/ubuntufs /bin/bash
```

### Implemented Concepts

- Isolation
    - PID namespace
    - UTS namespace
    - Mount namespace
- Filesystem
    - Root isolation using pivot_root
    - /proc mounted