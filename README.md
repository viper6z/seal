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

Terraform creates the AWS network and EC2 instance. Cloud-init installs Docker, clones the repository, builds the agent and installs its systemd service and timer.

The agent runs periodically and treats `compose.yaml` and `nginx/conf.d/` on `origin/main` as desired state. It:

1. fetches the latest commit;
2. stages the exact managed configuration;
3. validates Docker Compose and Nginx;
4. backs up and publishes the new files;
5. applies the Compose stack and recreates Nginx;
6. records the successfully applied commit;
7. restores the previous configuration if runtime apply fails.

GitHub Actions validates changes and uses AWS OIDC to run Terraform. Runtime deployment is owned by the host agent rather than pushed through CI.

## Repository layout

- `interface/`: Go CLI for manifest validation and configuration generation
- `agent/`: host reconciliation agent and systemd units
- `terraform/`: AWS infrastructure and cloud-init bootstrap
- `compose.yaml`: desired container runtime
- `nginx/conf.d/`: base and generated ingress configuration
- `.github/workflows/`: CI and Terraform deployment
- `docs/logbook.md`: project build-up and decisions

## Scope

Seal is intentionally not Kubernetes or a production platform. It is a focused learning project for understanding the layers between a developer-facing application definition and a reproducible running service on AWS.
