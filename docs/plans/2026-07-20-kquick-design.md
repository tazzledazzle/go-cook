# kquick CLI Design

## Goal

Add `kquick`, a read-only command-line tool that shortens common Kubernetes
Pod inspection workflows. Version one supports listing, describing, and reading
logs from Pods in an existing Kubernetes cluster.

## Scope

The first release provides:

- `kquick get pods`
- `kquick describe pod NAME`
- `kquick logs pod NAME`

Both `pod` and `pods` are accepted as resource aliases. Other Kubernetes
resources and all mutating operations are out of scope.

## Architecture

The executable lives at `design-kube/cmd/kquick`, alongside the existing API
server. Its `main` package only creates a signal-aware context, invokes the root
command, and maps failures to process exit status.

CLI construction and command handlers live under
`design-kube/internal/kquick`. Cobra handles commands, arguments, flags, usage,
and help. Kubernetes' typed `client-go` CoreV1 client handles Pod and Event API
operations.

Dependencies such as the Kubernetes client, log stream opener, and output
streams are injected into command handlers. This keeps cluster access separate
from presentation and allows tests to run without a live cluster.

## Configuration

Cluster credentials follow standard kubeconfig precedence:

1. `--kubeconfig`
2. `KUBECONFIG`
3. `~/.kube/config`

The global `--context` flag overrides the active kubeconfig context. The global
`--namespace` or `-n` flag overrides its namespace. If neither provides a
namespace, `default` is used.

## Commands

### Get Pods

`kquick get pods` prints a human-readable table by default. Columns are Pod
name, readiness, status, restart count, age, and node.

The `-o` flag accepts:

- `table` (default)
- `json`
- `yaml`

Structured formats emit Kubernetes Pod list objects and are suitable for
scripting.

### Describe Pod

`kquick describe pod NAME` prints a concise operational summary:

- Metadata and labels
- Phase, IP, and assigned node
- Containers, images, readiness, restart counts, and current states
- Pod conditions
- Recent events associated with the Pod

### Pod Logs

`kquick logs pod NAME` writes Pod logs directly to stdout. It supports:

- `-c`, `--container`
- `-f`, `--follow`
- `--tail`

When a Pod has multiple containers and no container is selected, the command
fails with a message listing valid container names.

## Errors and Cancellation

Argument, kubeconfig, and Kubernetes API errors include actionable context and
are written to stderr. Failed commands return a nonzero exit status.

The root command receives a signal-aware context. Ctrl-C cancels API requests
and active log streams.

## Testing

Tests are table-driven and cover:

- Command and argument validation
- Kubeconfig context and namespace precedence
- Table, JSON, and YAML output
- Pod summaries and event ordering
- API failures
- Single- and multi-container log selection
- Tail and follow options
- Context cancellation

The fake Kubernetes client is used for Pod and Event operations. Log streaming
uses an injected opener so tests can supply deterministic readers and errors.

Verification includes `gofmt`, `go vet ./...`, and `go test -race ./...` from
the `design-kube` module.

## Future Work

Possible later additions include Deployments and Services, context shortcuts,
diagnostic summaries, watch mode, and mutating commands. They are intentionally
excluded until the Pod-focused workflow is validated.
