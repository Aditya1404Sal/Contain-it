# Contain-it : A Mock Containerization Tool

# MVP Status :  not ready
TODO : Implement Root-fs extraction

This is a simple mock containerization tool written in Go. It mimics the behavior of basic container operations like pulling an image, unpacking it, and running commands within a new namespace. This tool does not fully replicate Docker's functionality but provides a lightweight alternative for educational purposes.

## Features

- Pull Docker images and save them as tar.gz archives.
- Unpack the archives and run commands within new namespaces.
- Use cgroups to limit resources.
- Chroot to the unpacked image root filesystem.

## Usage

### Commands

1. **run**: Run a command inside a new container.
   ```sh
   ./cnts run <image> <command>
   ```
   Example:
   ```sh
   ./cnts run ubuntu /bin/sh
   ```

2. **child**: This is an internal command used by `run`. It sets up the new namespaces and executes the command.

3. **pull**: Pull a Docker image and store it as a tar.gz archive.
   ```sh
   ./cnts pull <image>
   ```
   Example:
   ```sh
   ./cnts pull ubuntu
   ```

> **Note**: You need to run these commands as root:
> ```sh
> root@device:~# ./cnts run <image> <command>
> root@device:~# ./cnts pull <image>
> ```

### Pull Script

The `pull` script is a helper bash script to fetch Docker images, export their filesystem, and save it to the `assets` directory.

```sh
#!/bin/bash

set -e

defaultImage="hello-world"

image="${1:-$defaultImage}"
container=$(docker create "$image")

docker export "$container" -o "./assets/${image}.tar.gz" > /dev/null
docker rm "$container" > /dev/null

docker inspect -f '{{.Config.Cmd}}' "$image:latest" | tr -d '[]\n' > "./assets/${image}-cmd"

echo "Image content stored in assets/${image}.tar.gz"
```

## Building and Running

1. Build the Go program:
   ```sh
   go build -o cnts
   ```

2. Make the `pull` script executable:
   ```sh
   chmod +x pull
   ```

3. Use the commands as described above.

> **Note**: You need to run these commands as root:
> ```sh
> root@device:~# ./cnts run <image> <command>
> root@device:~# ./cnts pull <image>
> ```

## Note

This tool is for educational purposes and is not intended for production use. It demonstrates basic containerization concepts like namespaces and cgroups without the complexity and features of full-fledged container runtimes like Docker.
