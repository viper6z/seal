# Seal

Seal is a small single-host application platform I built to understand how infrastructure, deployment and reconciliation fit together before moving on to Kubernetes.

A developer describes an application in YAML. A Go CLI validates the manifest and generates Docker Compose and Nginx configuration. Git stores the desired state, while a Go agent on the EC2 host keeps the running system in sync with `main`.

```text
container image
→ application manifest
→ seal deploy
→ pull request
→ CI
→ merge to main
→ host reconciliation
→ Docker Compose
→ Nginx
→ application
```

## Application interface

```yaml
name: example-api
image: ghcr.io/example/example-api:v1
internal_port: 8080
exposure_type: public
allowed_public_routes:
  - /
  - /healthz
```

Public applications receive explicitly declared Nginx routes. Internal applications only join the backend Docker network and are not exposed publicly.

## How it works

The agent runs periodically and treats compose.yaml and nginx/conf.d/ on origin/main as desired state. It does not act on new commits alone: on every run it compares the target commit's blob hashes against the files on disk, and the services declared in Compose against the containers actually running. If either has diverged it reconciles, so a hand-edited config file or a stopped container is corrected without waiting for a commit.

When it reconciles, the agent:

stages the exact managed configuration from the target commit;
validates Docker Compose and Nginx against the staged files;
backs up the current configuration and publishes the new files atomically;
applies the Compose stack and recreates Nginx;
records the successfully applied commit;
restores the previous configuration and runtime if the apply fails.

## Repository layout

- `interface/`: Go CLI for manifest validation and configuration generation
- `agent/`: host reconciliation agent and systemd units
- `terraform/`: AWS infrastructure and cloud-init bootstrap
- `compose.yaml`: desired container runtime
- `nginx/conf.d/`: base and generated ingress configuration
- `.github/workflows/`: CI and Terraform deployment
- `docs/logbook.md`: project build-up and decisions

## Scope

Seal is single-host by design and is not a production platform. Within that scope it does the real work: atomic configuration publishing with rollback, level-triggered reconciliation against observed state, and infrastructure provisioned through Terraform with OIDC-authenticated CI. It was built to understand the layers between a developer-facing application definition and a reproducible running service on AWS.


