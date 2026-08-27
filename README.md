# kubectl-meta

A kubectl plugin that displays the `.metadata` block of any Kubernetes resource.

## Installation

Download the binary for your platform from the [GitHub Releases](../../releases) page and place it on your `PATH` as `kubectl-meta`:

```bash
# Example for Linux amd64
curl -Lo kubectl-meta https://github.com/nickytd/kubectl-meta-plugin/releases/latest/download/kubectl-meta_linux_amd64
chmod +x kubectl-meta
mv kubectl-meta /usr/local/bin/
```

Or build from source:

```bash
git clone https://github.com/nickytd/kubectl-meta-plugin.git
cd kubectl-meta-plugin
make install   # builds and copies bin/kubectl-meta to $GOPATH/bin
```

## Usage

```bash
kubectl meta TYPE/NAME [flags]
```

```bash
kubectl meta pod/my-pod
kubectl meta deploy/my-deploy -n kube-system
kubectl meta my-pod                        # bare name defaults to pod
kubectl meta deploy/my-deploy -o json
kubectl meta pod/my-pod --with-managed-fields
kubectl meta node/my-node                  # cluster-scoped resources work too
kubectl meta namespace/kube-system
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output` | `yaml` | Output format: `yaml` or `json` |
| `--with-managed-fields` | `false` | Include `managedFields` in output |
| `-n, --namespace` | current context | Namespace |
| `--context` | current context | Kubeconfig context |
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig |

## Shell completion

```bash
# zsh (one-time)
kubectl-meta completion zsh > "${fpath[1]}/_kubectl-meta"
exec zsh

# current session only
source <(kubectl-meta completion zsh)

# bash
source <(kubectl-meta completion bash)

# fish
kubectl-meta completion fish | source

# PowerShell
Add-Content $PROFILE "; kubectl-meta completion powershell | Out-String | Invoke-Expression"
```

## Development

### Prerequisites

- Go 1.26+
- Access to a Kubernetes cluster (`kubectl` configured)

### Build

```bash
make        # compile to bin/kubectl-meta (runs fmt + gci + lint + license first)
```

### All make targets

| Target | Description |
|--------|-------------|
| `make build` | compile (runs check first) — default target |
| `make check` | fmt + import ordering + lint + license headers |
| `make lint` | run golangci-lint |
| `make fmt` | gofmt + goimports |
| `make tidy` | go mod tidy for main and tools modules |
| `make govulncheck` | vulnerability scan |
| `make clean` | remove `bin/` and `dist/` |

### Module structure

- `go.mod` — plugin dependencies only
- `tools/go.mod` — isolated tool dependencies (`golangci-lint`, `addlicense`, `gci`, `govulncheck`)

## Release

Tag the commit and push:

```bash
git tag v0.1.0
git push origin main
git push origin v0.1.0
```

[goreleaser](https://goreleaser.com) builds binaries for all platforms and publishes the GitHub Release with release notes generated from the commit history.

## License

Apache-2.0 — see [LICENSE](LICENSE).