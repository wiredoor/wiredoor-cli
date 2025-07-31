<p align="center"> <img src="https://www.wiredoor.net/images/wiredoor.svg" alt="Wiredoor logo" width="60" /> </p>

<h1 align="center" style="color:#1c398e">
  Wiredoor
</h1>

<p align="center">
  <strong>Expose private services securely through reverse VPN tunnel powered by WireGuard and NGINX.</strong><br />
  Open-source | Self-hosted
</p>

<p align="center">
  <a href="https://www.wiredoor.net/docs">Documentation</a> •
  <a href="https://github.com/wiredoor/wiredoor">Core Server</a> •
  <a href="https://github.com/wiredoor/wiredoor-cli">CLI</a> •
  <a href="https://charts.wiredoor.net">Helm Charts</a>
</p>


<p align="center">
  <a href="https://github.com/wiredoor/wiredoor-cli/actions">
    <img src="https://github.com/wiredoor/wiredoor-cli/actions/workflows/release.yml/badge.svg" alt="CI Status" />
  </a>
  <a href="https://github.com/wiredoor/wiredoor/releases">
    <img src="https://img.shields.io/github/v/release/wiredoor/wiredoor?label=Wiredoor" alt="Wiredoor Release" />
  </a>
  <a href="https://github.com/wiredoor/wiredoor-cli/releases">
    <img src="https://img.shields.io/github/v/release/wiredoor/wiredoor-cli?label=CLI&color=silver" alt="Wiredoor CLI Release" />
  </a>
  <a href="https://github.com/wiredoor/wiredoor-cli">
    <img src="https://img.shields.io/github/license/wiredoor/wiredoor-cli" alt="License" />
  </a>
  <a href="https://ghcr.io/wiredoor/wiredoor">
    <img src="https://img.shields.io/badge/GHCR-wiredoor-blue?logo=github" alt="Wiredoor GHCR" />
  </a>
  <a href="https://ghcr.io/wiredoor/wiredoor-cli">
    <img src="https://img.shields.io/badge/GHCR-gateway-silver?logo=github" alt="Wiredoor Docker Gateway GHCR" />
  </a>
  <a href="https://github.com/wiredoor/wiredoor-cli/stargazers">
    <img src="https://img.shields.io/github/stars/wiredoor/wiredoor-cli?style=social" alt="GitHub Stars" />
  </a>
</p>

---

# Wiredoor CLI Command Reference

[Wiredoor CLI](https://github.com/wiredoor/wiredoor-cli)  is the command-line interface for interacting with the Wiredoor platform. 
It provides a simple and secure way to register nodes, expose services, and manage connectivity directly from the terminal.

It allows you to:

- Easily connect to new node in Wiredoor using admin credentials.
- Quickly connect a private machine to the Wiredoor network without complex configuration.
- Expose HTTP or TCP services running on your local/private network with just one command.
- Enable or disable services on the fly
- Check VPN status and connection logs
- Automate service exposure on boot
- Authenticate via token-based access

This CLI is especially useful for nodes running in headless environments (e.g., servers, containers, Raspberry Pi, etc.).

## Installation

You can use the auto-installer:

```bash
curl -s https://www.wiredoor.net/install-wiredoor-cli.sh | sh
```

Or download a package from [GitHub Releases](https://github.com/wiredoor/wiredoor-cli/releases).

## Command reference

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

### Connect to Wiredoor Node

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

### Wiredoor disconnect

Stop the active VPN tunnel and disable all services (temporarily).

```bash
wiredoor disconnect
```

- Does **not** delete the node configuration
- Use before maintenance or to restart

## Systemd service

If installed via package, Wiredoor includes a `systemd` service that runs a health-check in background to ensure persistent connectivity:

- Reconnects automatically if connection drops
- Verifies service exposure health every 15 seconds
- Ideal for long-running nodes and production servers

To run wiredoor service start and enable `wiredoor.service`:

Systemd based systems:

```bash
sudo systemctl enable wiredoor
sudo systemctl start wiredoor
```

OpenRC based systems (Alpine Linux):

```bash
chmod +x /etc/init.d/wiredoor
rc-update add wiredoor
rc-service wiredoor start
```

**This keeps the node online across reboots and network changes.**
