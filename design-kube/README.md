# Design-Kube

## What This Is

A Kubernetes-style container orchestration platform built from scratch. It orchestrates containers across many nodes with self-healing and declarative desired state, exposes an API-first control plane with extensibility (CRDs/controllers), and provides multi-tenant soft isolation (namespaces, quotas). It is for operators and platform teams who need a custom orchestrator with full control over architecture and trade-offs.

## Core Value

Declarative desired state across many nodes with self-healing and an API-first, extensible control plane; zero data loss on state changes and control-plane HA. If everything else fails, the system must maintain desired state and recover workloads.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Orchestrate containers across many nodes with declarative desired state
- [ ] API server (REST/gRPC, OpenAPI) with authn/authz, admission, validation
- [ ] etcd as state store with watch/list and transactional writes
- [ ] Controllers (e.g. RC, Deploy, HPA, GC, CRD) with work queues and reconciliation
- [ ] Scheduler (binpack/priority) binding PodSpecs to nodes
- [ ] Node agents (kubelet-style): pod lifecycle, probes, CRI (containerd/runc), CNI, CSI
- [ ] Service discovery and ingress (L4 Service VIP; Envoy/Gateway API direction)
- [ ] Observability: metrics, traces, logs (OTel)
- [ ] RBAC, namespaces, quotas; NetworkPolicies; admission webhooks (policy)
- [ ] CRDs and custom controllers for extensibility
- [ ] Control-plane HA; P95 pod create latency < 2s; scale 5k+ nodes, ~100–300 pods/node

### Out of Scope

- Docker as CRI — deprecated for prod control plane; use containerd
- Legacy iptables-only kube-proxy for large clusters — prefer eBPF (e.g. Cilium)
- Global strong-consistency DB (Spanner/Cockroach) for state — etcd sufficient unless global multi-region requirement
- Hard multi-tenant isolation in v1 — soft isolation (namespaces, quotas, NetworkPolicies) first
- Multi-cluster federation in v1 — cluster-per-env or later KubeFed/management plane

## Context

- **Architecture:** API Server ↔ etcd; controllers and scheduler consume watch/list; scheduler binds pods to nodes; node agents (kubelet) run pods via CRI/CNI/CSI; observability and service VIP (kube-proxy/eBPF) on nodes.
- **Trade-off picks:** etcd (state), containerd (CRI), eBPF for networking (e.g. Cilium), Envoy/Gateway API for ingress, network block (EBS/PD-style) for default PVs.
- **SLOs:** API Server 99.95% availability, P95 CRUD < 100 ms; pod start P95 < 2s (image cached); 5k+ nodes, ~100–300 pods/node.
- **To clarify (Section 12):** Target scale (nodes, pods, tenants); cloud vs on-prem vs hybrid (affects CNI/CSI); required isolation strength; compliance (audit, logging retention); upgrade/ops philosophy (managed vs self-hosted). These will tune configs and concrete tech choices.

## Constraints

- **Tech stack:** etcd, containerd, eBPF direction for networking — already decided from trade-off analysis.
- **Performance:** P95 pod create < 2s; node scale 5k+; pods per node ~100–300.
- **Reliability:** Control-plane HA; zero data loss on state changes.
- **Security:** TLS everywhere; mTLS kubelet↔apiserver; RBAC, PSA/PodSecurity, admission webhooks.

## Key Decisions

| Decision                                     | Rationale                                                   | Outcome   |
|----------------------------------------------|-------------------------------------------------------------|-----------|
| etcd for state store                         | Battle-tested, watch semantics, txn; de facto for K8s-style | — Pending |
| containerd for CRI                           | CNCF, stable, fast; Docker deprecated as CRI                | — Pending |
| eBPF for Service/networking                  | High perf, observability; iptables rule explosion at scale  | — Pending |
| Envoy/Gateway API for ingress                | Modern, extensible; long-term direction                     | — Pending |
| Network block (EBS/PD-style) for default PVs | Easy, managed; CSI for Ceph/Rook when on-prem               | — Pending |
| Binpack/spread/priority scheduler strategies | Configurable by use case (cost vs HA vs fairness)           | — Pending |

---
*Last updated: 2025-02-16 after initialization*
