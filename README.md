# Wiredoor CLI Command Reference

`wiredoor-cli` is a lightweight command-line client written in Go, designed to interact with the Wiredoor Server.

It allows you to:

- Connect your node to the VPN
- Expose or unexpose services
- Check VPN status and connection logs
- Automate service exposure on boot
- Authenticate via token-based access

This CLI is especially useful for nodes running in headless environments (e.g., servers, containers, Raspberry Pi, etc.).

## Installation

You can install wiredoor-cli from prebuilt packages or use the multi-arch Docker image. 
Pick the method that suits your platform from the [Releases](https://github.com/wiredoor/wiredoor-cli/releases) page:

### Debian/Ubuntu

```bash
wget https://github.com/wiredoor/wiredoor-cli/releases/download/v1.0.0/wiredoor_1.0.0-1_debian_amd64.deb
sudo apt install -f ./wiredoor_1.0.0-1_debian_amd64.deb
```

### RHEL / Fedora / CentOS / AlmaLinux

```bash
wget https://github.com/wiredoor/wiredoor-cli/releases/download/v1.0.0/wiredoor_1.0.0-1_rpm_amd64.rpm
sudo dnf install -y wiredoor_1.0.0-1_rpm_amd64.rpm
```

### Alpine Linux

```bash
wget https://github.com/wiredoor/wiredoor-cli/releases/download/v1.0.0/wiredoor_1.0.0-1_alpine_amd64.apk
sudo apk add --allow-untrusted wiredoor_1.0.0-1_alpine_amd64.apk
```

### Login and Create Node

The fastest way to onboard a new system as a node. Authenticate with a Wiredoor server using admin credentials and register this node via interactive prompts.

```bash
wiredoor login --url https://your-wiredoor-instance-or-ip
```

- Prompts for your admin credentials (email/password)
- Prompts you to assign a name to the node. Defaults to the machine’s hostname.
- Prompts you if you want to make this a gateway.
- Network configuration for gateway and traffic.

This command retrieves and saves the node’s token to `/etc/wiredoor/config.ini` and connect to wiredoor server.

### Connect

Establish a VPN connection using saved or provided credentials.

```bash
wiredoor connect
wiredoor connect --url=https://wiredoor.example.com --token=ABC123
```

- Uses `/etc/wiredoor/config.ini` by default
- Optionally override `--url` and `--token`

### Wiredoor config

Write the server URL and token to the config file without connecting.

```bash
wiredoor config --url=https://wiredoor.example.com --token=ABC123
```

- Saves config to `/etc/wiredoor/config.ini`
- Does **not** start the connection

### Wiredoor Expose HTTP Service

Expose a local HTTP service via Wiredoor.

```bash
wiredoor http my-website --domain website.com --port 3000
```

Optional flags:

- `--path /ui` (default: /)
- `--proto https` (default: http)
- `--backendHost` (useful if acting as a gateway)
- `--allow` / `--block` for IP access control

### Wiredoor Expose TCP Service

Expose a generic TCP/UDP service via wiredoor available port.

```bash
wiredoor tcp ssh-access --port 22
wiredoor tcp redis --port 6379 --ssl --allowedIps 192.168.0.0/24
```

Optional flags:

- `--proto tcp|udp` (default: `tcp`)
- `--ssl` wrap in TLS
- `--backendHost` (useful if acting as a gateway)
- `--allow` / `--block` for IP access control

### Wiredoor Status

View current VPN and service status.

```bash
wiredoor status
wiredoor status --health
wiredoor status --watch --interval 10
```

Flags:

- `--health`: Health check (exit 0/1)
- `--watch`: Continuous monitoring
- `--interval`: Poll interval (default: 5s)

### Wiredoor disconnect

Stop the active VPN tunnel and disable all services (temporarily).

```bash
wiredoor disconnect
```

- Does **not** delete the node configuration
- Use before maintenance or to restart

### Wiredoor disable

Temporarily disable an exposed service by name.

```bash
wiredoor disable http my-website
wiredoor disable tcp db-access
```

- Blocks public access
- Use `wiredoor enable` to restore

### Wiredoor enable

Re-enable a previously disabled service.

```bash
wiredoor enable http my-website
wiredoor enable tcp db-access
```

- Restores service availability
- Requires existing configuration




