# PlatformX FinOps

![CI](https://img.shields.io/github/actions/workflow/status/your-username/platformx-finops/ci.yml?label=CI)
![License](https://img.shields.io/badge/license-MIT-blue)
![Kubernetes](https://img.shields.io/badge/kubernetes-1.29-blue)
![Go](https://img.shields.io/badge/go-1.22-teal)

A production-grade FinOps platform on AWS EKS — featuring a custom Kubernetes operator,
ML cost forecasting (Prophet), WebAssembly OPA policies, and full GitOps with ArgoCD.

## Architecture

```
┌─────────────────────────────────────────┐
│         Developer Portal (Backstage)    │
├─────────────────────────────────────────┤
│      CI/CD — GitHub Actions + ArgoCD   │
├─────────────────────────────────────────┤
│   Kubernetes EKS + Custom Operator      │
├─────────────────────────────────────────┤
│   FinOps Core — Collector · ML · Alerts │
├─────────────────────────────────────────┤
│   Infra AWS — Terraform + Crossplane    │
├─────────────────────────────────────────┤
│   Security — Kyverno · Vault · WASM     │
├─────────────────────────────────────────┤
│   Observability — OTel · Grafana · Loki │
└─────────────────────────────────────────┘
```

## Services

| Service | Language | Description |
|---|---|---|
| `data-collector` | Go | Polls AWS Cost Explorer, Prometheus, Kubecost |
| `cost-analyzer` | Go | Cost breakdown by team/namespace/env |
| `recommendation-engine` | Go | Right-sizing, Spot recommendations |
| `anomaly-detection` | Go | Z-score anomaly detection on cost timeseries |
| `budget-alert` | Go | Budget thresholds + Slack/PagerDuty alerts |
| `ml-forecasting` | Python | Prophet-based 30-day cost forecasting |
| `operator` | Go | Custom Kubernetes operator — FinOpsPolicy CRD |
| `frontend` | React | Real-time FinOps dashboard |

## Quickstart (local dev)

```bash
# Prerequisites: Docker, Docker Compose, Go 1.22+, Python 3.11+
git clone https://github.com/your-username/platformx-finops
cd platformx-finops

# Start local stack
make dev-up

# Seed fake data (1 year of cost history)
make seed-data

# Run all services
make run-all
```

## Repository structure

```
platformx-finops/
├── services/
│   ├── data-collector/
│   ├── cost-analyzer/
│   ├── recommendation-engine/
│   ├── anomaly-detection/
│   ├── budget-alert/
│   └── ml-forecasting/
├── operator/              # Custom Kubernetes operator
├── frontend/              # React dashboard
├── infra/
│   ├── terraform/         # AWS infra modules
│   └── terragrunt/        # Multi-env config
├── gitops/
│   ├── argocd/            # ArgoCD apps
│   └── helm/              # Helm charts
├── policies/
│   ├── rego/              # OPA policies
│   └── wasm/              # Compiled .wasm bundles
├── docs/adrs/             # Architecture Decision Records
└── .github/workflows/     # CI/CD pipelines
```
