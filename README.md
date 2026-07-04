# Seal

Seal is a small AWS deployment platform lab.

The point is to build and understand a real path from a Git repository to containerised workloads running on an AWS VM.

```text
Git repository
→ GitHub Actions
→ Terraform
→ AWS EC2
→ Docker Compose
→ Nginx
→ workload
```

This is mainly a learning project, but I am trying to make it one coherent system instead of just collecting tools.

I want to understand what every layer does, what problem it solves, and what actually happens when something breaks.

## What Seal does

```text
Pull request
→ CI validates Terraform and Docker Compose

Merge to main
→ GitHub Actions assumes an AWS role through OIDC
→ Terraform reconciles AWS infrastructure
→ GitHub Actions sends a deployment command through SSM
→ the VM pulls the latest repository revision
→ Docker Compose rebuilds and reconciles the stack
→ Nginx serves the intended public HTTP routes
```

The VM is configured on first boot through cloud-init.

```text
Terraform
→ creates AWS infrastructure

cloud-init
→ installs Docker, Docker Compose, Git, and prepares the host

GitHub Actions + SSM
→ deploys later repository changes to the existing VM

Docker Compose
→ defines and runs the workloads

Nginx
→ acts as the public HTTP boundary
```

## Current architecture

```text
GitHub

pull request
├── CI
│   ├── terraform fmt
│   ├── terraform validate
│   └── docker compose config
│
└── merged pull request
    └── CD
        ├── GitHub OIDC login to AWS
        ├── Terraform plan
        ├── Terraform apply
        ├── AWS Systems Manager
        └── docker compose up -d --build --remove-orphans


AWS

VPC
└── public subnet
    ├── internet gateway
    ├── route table
    ├── security group
    │   ├── inbound TCP/80 from anywhere
    │   └── all outbound traffic
    │
    └── Ubuntu EC2 VM
        ├── cloud-init bootstrap
        ├── Docker Engine
        ├── Docker Compose
        ├── Git
        └── SSM agent


Docker Compose

edge network
└── nginx

backend network
├── nginx
├── homelab-api
├── tcp-service
├── udp-service
└── toolbox
```

## Public ingress

Only TCP port `80` is publicly reachable.

```text
public internet
→ AWS security group
→ EC2 host port 80
→ Nginx
→ backend Docker network
→ explicitly allowed API route
```

Nginx is the only service connected to both the `edge` and `backend` Docker networks.

The currently exposed routes are explicitly allow-listed:

```text
GET /
GET /health
GET /time
```

Anything else stops at Nginx:

```text
GET /random
→ Nginx 404
→ API never receives the request
```

The Flask API, TCP service, and UDP service do not publish host ports directly.

Management of the VM is done through AWS Systems Manager rather than public SSH.

## Workloads

| Service | What it is |
|---|---|
| `homelab-api` | Small Flask API used to test the HTTP deployment path |
| `nginx` | Public HTTP entry point and reverse proxy |
| `tcp-service` | Small custom TCP protocol experiment |
| `udp-service` | Small UDP live-text synchronisation experiment |
| `toolbox` | Internal container used to test TCP and UDP services |

The API is reachable internally through Docker Compose service discovery:

```text
http://homelab-api:5000
```

The TCP and UDP services are private to the backend Docker network.

## Adding a new workload

Seal does not have a generator yet.

The current golden path is:

```text
1. Add a folder for the application
2. Give it its own Dockerfile
3. Add a Compose service definition
4. Include that service from the root Compose file
5. Put it on the backend network
6. Keep it private by default
7. Add an explicit Nginx route only when it should be public
8. Open a pull request and merge it
```

A deployed service is not automatically public.

```text
new Docker service
→ private by default

explicit Nginx route
→ public HTTP access
```

## Running locally

Start the stack from the repository root:

```bash
docker compose up --build
```

Check the running services:

```bash
docker compose ps
```

Test the HTTP boundary:

```bash
curl -i http://localhost/health
curl -i http://localhost/random
```

Expected result:

```text
/health
→ 200 from the API through Nginx

/random
→ Nginx 404
```

Stop the stack:

```bash
docker compose down
```

## Testing the protocol services

Start the TCP client:

```bash
docker compose exec -it toolbox python tcp_client.py
```

Start the UDP client:

```bash
docker compose exec -it toolbox python udp_client.py
```

Run the UDP client from two terminals to see one client update the other.

## Terraform

Terraform is split into two roots:

```text
terraform/bootstrap
→ long-lived setup such as remote state, IAM, and GitHub OIDC trust

terraform
→ Seal infrastructure: VPC, subnet, security group, EC2 VM, and cloud-init
```

The main Terraform state is stored remotely in S3 so local Terraform and GitHub Actions use the same state.

The main infrastructure includes:

```text
VPC
→ public subnet
→ internet gateway
→ route table
→ security group
→ EC2 key pair
→ Ubuntu EC2 VM
```

## Repository layout

```text
.
├── .github/
│   └── workflows/
│       ├── ci.yaml
│       └── cd.yaml
│
├── docs/
│   └── logbook.md
│
├── homelab-api/
├── nginx/
│   ├── compose.yaml
│   └── conf.d/
│       └── nginx.conf
│
├── tcp-service/
├── toolbox/
├── udp-service/
│
├── terraform/
│   ├── bootstrap/
│   ├── cloud-init.yaml
│   ├── network.tf
│   └── terraform.tf
│
└── compose.yaml
```

## Things learned so far

```text
A container port and a host port are different things.

127.0.0.1 inside a container means that container itself.

Docker Compose service names act as internal DNS names.

TCP is a byte stream, so application protocols need framing.

UDP preserves message boundaries but does not guarantee delivery or ordering.

A public VPC route and a public security-group ingress rule solve different problems.

Terraform state is part of the deployment system.

OIDC lets GitHub Actions use temporary AWS credentials instead of stored AWS keys.

First-boot host setup and recurring workload deployment are different problems.

A service existing in Docker does not mean it is publicly exposed.

Nginx can act as an explicit HTTP boundary where routes stay private until they are intentionally exposed.
```

## Not included yet

Seal v0.1 is a small first version, not a production platform.

Things deliberately left for later:

```text
Domain name and TLS
Container registry
Application test suite
Monitoring and observability
Alerting
Rate limiting
WAF
Load balancer
Multiple environments
Multiple hosts
Kubernetes
```

Kubernetes will be a separate lab rather than something forced into this Docker Compose project.

## Notes

The full build-up of the project is documented in [docs/logbook.md](docs/logbook.md).

Seal is still evolving, but v0.1 is the first complete shape of the system:

```text
Git repository
→ GitHub Actions
→ Terraform
→ AWS EC2
→ SSM deployment
→ Docker Compose
→ Nginx
→ containerised workloads
```
