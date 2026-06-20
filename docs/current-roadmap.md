# Azure platform homelab roadmap

## Lab purpose

Build a private-by-default platform lab around a Python API, Docker Compose, and Nginx. First run it locally, then deploy it to an Azure VM, recreate the infrastructure with Terraform, configure it with Ansible, and validate it through GitHub Actions.

The goal is not to collect tools. The goal is to understand the full request and deployment path:

```text
Client
→ DNS or local hostname mapping
→ Nginx reverse proxy
→ private Docker network
→ Python API
```

Later:

```text
Terraform
→ Azure infrastructure

Ansible
→ VM configuration

GitHub Actions
→ testing and deployment validation
```

---

# Phase 1 — Programming language: Python

**roadmap.sh section:** Programming Language
**Relevant project cards:** Server Performance Stats, Log Archive Tool, Nginx Log Analyzer

## Goal

Use Python for small operational tools as well as the Flask API.

## Build

Keep extending the existing `homelab-api` rather than making disconnected projects.

Add endpoints:

```text
/health
/time
/info
```

Example responsibilities:

```text
/health  → confirms the app is running
/time    → shows uptime and current time
/info    → shows app version, hostname, and environment
```

Create a `scripts/` folder:

```text
scripts/
├── server_stats.py
├── archive_logs.py
└── nginx_log_analyzer.py
```

### Project: Server Performance Stats

Write a Python or Bash script that reports:

```text
CPU usage
memory usage
disk usage
running Docker containers
container status
```

Run it locally first. Later run it on the Azure VM.

### Project: Log Archive Tool

Write a script that:

```text
takes a log directory
compresses older logs
adds the current date to the archive name
keeps a configurable number of archives
```

Do this after Nginx exists, so you have real access logs to work with.

### Project: Nginx Log Analyzer

Write a simple CLI tool that reads Nginx access logs and answers:

```text
most requested route
top response codes
number of requests
most common client IPs
```

## Done when

```bash
pytest
python scripts/server_stats.py
python scripts/nginx_log_analyzer.py ./logs/access.log
```

all work from a clean clone.

---

# Phase 2 — Operating system: Linux

**roadmap.sh section:** Operating System
**Relevant project cards:** Dummy Systemd Service, SSH Remote Server Setup

## Goal

Become comfortable operating Linux rather than only running Docker commands.

## Learn and practise

```text
filesystem navigation
permissions and ownership
processes
ports
system logs
environment variables
package management
SSH
```

Use WSL now. Use Ubuntu on Azure later.

Useful commands to understand:

```bash
pwd
ls
cd
cat
grep
find
chmod
chown
ps
top
ss
curl
journalctl
systemctl
```

### Project: Dummy Systemd Service

Do this only after you have an Azure VM.

Create a tiny script that writes a timestamp to a log file every minute. Run it as a systemd service.

The point is not the script. The point is learning:

```text
service definitions
starting and stopping services
checking status
reading logs with journalctl
```

## Done when

You can answer:

```text
What process is using a port?
Where are the logs?
How do I check whether a service starts after reboot?
```

---

# Phase 3 — Networking and protocols

**roadmap.sh section:** Networking & Protocols
**Relevant project cards:** Basic DNS Setup, SSH Remote Server Setup

## Goal

Understand how a request reaches the correct application.

## Learn through your current lab

```text
IP addresses
loopback: 127.0.0.1
ports
DNS
hosts file mappings
HTTP requests
Host headers
reverse proxies
private Docker networks
SSH
```

Your local request flow should be documented as:

```text
Browser requests:
http://api.lab.home/time

Windows hosts file:
api.lab.home → 127.0.0.1

Browser connects:
127.0.0.1:80

Nginx receives:
Host: api.lab.home

Nginx forwards:
homelab-api:5000

Flask handles:
/time
```

### Project: Basic DNS Setup

For now, this is deliberately small:

```text
127.0.0.1 api.lab.home
127.0.0.1 home.lab.home
```

in the Windows hosts file.

Do not build a DNS server yet. The objective is to understand hostname-to-IP mapping, not operate DNS infrastructure.

Later, private Azure DNS becomes an optional networking phase.

## Done when

You can explain the difference between:

```text
DNS:
api.lab.home → machine IP

Nginx:
Host: api.lab.home → application container

Docker networking:
homelab-api → container IP and port 5000
```

---

# Phase 4 — Docker

**roadmap.sh section:** Docker
**Relevant project cards:** Basic Dockerfile, Multi-Container Application, Dockerized Service

## Goal

Run your app as a repeatable containerised stack.

## Project: Basic Dockerfile

You have already completed the foundation:

```text
Python API
requirements.txt
Dockerfile
docker build
docker run
```

Improve it by adding:

```text
health endpoint
health check
environment variables
basic logging
```

## Project: Multi-Container Application

Replace Traefik with Nginx.

Your stack becomes:

```text
Nginx
↓ private Docker network
homelab-api
```

Important design rule:

```text
Nginx exposes port 80 to the host.

The API does not expose port 5000 to the host.
```

Architecture:

```text
Browser
→ localhost:80
→ Nginx
→ homelab-api:5000
```

## Project: Static Site Server

Add a simple static landing page served by Nginx.

```text
http://home.lab.home
```

It can contain:

```text
Homelab status
links to available services
short description of the lab
```

This gives you two Nginx behaviours:

```text
home.lab.home → static HTML
api.lab.home  → reverse proxy to Flask
```

## Done when

```bash
docker compose up -d
```

starts the entire stack and:

```text
http://home.lab.home
http://api.lab.home/time
```

both work.

Also confirm this does not work:

```text
http://localhost:5000
```

That proves Nginx is the only exposed front door.

---

# Phase 5 — Nginx

**roadmap.sh section:** Nginx
**Relevant project cards:** Static Site Server, Nginx Log Analyzer

## Goal

Understand reverse proxy configuration explicitly.

## Learn

```text
server blocks
listen directives
server_name
location blocks
proxy_pass
access logs
error logs
configuration validation
```

Your Nginx configuration should explicitly express:

```text
Requests for api.lab.home
→ forward to homelab-api:5000

Requests for home.lab.home
→ serve static files
```

Practise diagnosing failures:

```text
wrong server_name
wrong upstream port
API container stopped
Nginx syntax error
Docker network mismatch
```

Use:

```bash
nginx -t
docker compose logs nginx
docker compose logs homelab-api
```

## Done when

You can deliberately break the proxy, find the cause, fix it, and explain why it failed.

---

# Phase 6 — Git and GitHub

**roadmap.sh section:** Git and GitHub

## Goal

Treat the lab like a small real platform project.

## Repository structure

```text
homelab/
├── app/
├── nginx/
├── scripts/
├── docs/
├── infra/
│   └── terraform/
├── config/
│   └── ansible/
└── .github/
    └── workflows/
```

## Work style

Use branches for meaningful changes:

```text
feature/health-endpoint
feature/nginx-proxy
feature/static-homepage
feature/azure-vm
feature/terraform-infra
```

Each pull request should state:

```text
What changed
Why it changed
How it was tested
```

Keep out of Git:

```text
.env
private keys
tokens
real IP addresses
Terraform state files
```

## Done when

A stranger can clone the repository, read the README, and understand:

```text
what the lab does
how the request flows
how to run it locally
what each folder contains
```

---

# Phase 7 — Cloud provider: Azure

**roadmap.sh section:** AWS
**Your translation:** Azure
**Relevant project cards:** EC2 Instance, SSH Remote Server Setup

## Goal

Deploy the same Docker Compose stack to a real Linux server.

## Azure equivalent mapping

```text
AWS VPC              → Azure VNet
AWS Subnet           → Azure Subnet
Security Group       → Network Security Group
EC2                  → Azure Virtual Machine
IAM                  → Microsoft Entra ID, RBAC, Managed Identity
CloudWatch           → Azure Monitor and Log Analytics
S3                   → Azure Storage Account
```

## Manual Azure deployment first

Create manually:

```text
resource group
VNet
subnet
network security group
Ubuntu VM
SSH key access
```

Security posture:

```text
Allow SSH only from your current public IP.
Do not expose HTTP, HTTPS, or port 5000 publicly.
```

Install Docker, copy or clone the repository, and run the stack.

Test from inside the VM:

```bash
curl -H "Host: api.lab.home" http://localhost/time
```

## Project: SSH Remote Server Setup

Learn:

```text
SSH keys
known_hosts
copying files
Git clone over SSH
remote Docker commands
basic VM hardening
```

## Done when

You can:

```text
SSH into the VM
start the Compose stack
inspect logs
test Nginx internally
stop the VM when you are done
```

---

# Phase 8 — Terraform

**roadmap.sh section:** Terraform
**Relevant project card:** IaC on DigitalOcean
**Your translation:** IaC on Azure

## Goal

Recreate the Azure environment from code.

## Terraform should create

```text
resource group
VNet
subnet
network security group
network interface
public IP for restricted SSH access
Ubuntu VM
```

Start simple. Do not introduce modules immediately.

Use a folder structure like:

```text
infra/terraform/
├── main.tf
├── variables.tf
├── outputs.tf
├── providers.tf
└── terraform.tfvars.example
```

## Required workflow

```bash
terraform fmt
terraform validate
terraform plan
terraform apply
terraform destroy
```

## Important milestone

Build manually once.

Then delete the environment.

Then recreate it with Terraform.

That is when Terraform becomes real rather than theoretical.

## Done when

You can run:

```bash
terraform apply
```

and get a usable Azure VM without clicking through the Azure Portal.

---

# Phase 9 — Ansible

**roadmap.sh section:** Ansible
**Relevant project card:** Configuration Management

## Goal

Separate infrastructure creation from server configuration.

```text
Terraform creates Azure resources.

Ansible configures the Linux VM.
```

## Ansible should do

```text
update packages
install Docker
install Docker Compose plugin
create deployment directory
copy or clone repository
start Docker Compose
```

Later it can also configure:

```text
firewall rules
log rotation
system users
SSH hardening
monitoring agent
```

## Required design principle

The playbook should be safe to run repeatedly.

```text
Run once → configure VM
Run again → no unnecessary changes
```

## Done when

This is your rebuild flow:

```text
terraform apply
→ ansible-playbook
→ Docker Compose stack starts
→ Nginx routes to the API
```

---

# Phase 10 — GitHub Actions

**roadmap.sh section:** GitHub Actions
**Relevant project card:** Node.js Service Deployment
**Your translation:** Python Flask service deployment

## Goal

Start with CI. Do not automate deployment before you trust your tests and builds.

## Pull request workflow

Run on every pull request:

```text
Python tests
Docker image build
docker compose config
Nginx configuration validation
terraform fmt -check
terraform validate
```

## Later deployment workflow

After the Azure VM, Terraform, and Ansible phases are stable:

```text
push to main
→ GitHub Actions runs checks
→ approved deployment step
→ VM receives updated configuration or application version
```

Keep deployment simple initially. The first goal is proof that the pipeline works, not “perfect enterprise CI/CD.”

## Done when

A pull request cannot be merged until the project builds and validates successfully.

---

# Phase 11 — Monitoring, logs, and reliability

**roadmap.sh section:** Monitoring
**Relevant project cards:** Simple Monitoring, Server Performance Stats, Nginx Log Analyzer, Automated DB Backups

## Goal

Know whether the platform is healthy without guessing.

## Start small

Add:

```text
Docker health checks
Nginx access and error logs
application logs
restart policies
server stats script
```

Then add one monitoring tool:

```text
Netdata
or
Azure Monitor
```

Do not add Prometheus and Grafana just because they are popular. Add them when you have useful metrics to inspect.

Useful metrics:

```text
CPU usage
memory usage
disk space
container restarts
API response time
HTTP status codes
request volume
```

## Database backups

Do not do this until you add a real database.

When you eventually add PostgreSQL, create:

```text
scheduled backup
backup retention policy
restore test
```

A backup is only useful if you have tested restoring it.

## Done when

You can answer:

```text
Is the VM healthy?
Is Nginx receiving requests?
Is the API healthy?
Did any container restart?
Are errors increasing?
```

---

# Phase 12 — Optional private Azure networking

**roadmap.sh section:** Advanced networking follow-up

## Goal

Learn private access without making your services public.

Only begin this after the Azure VM and Terraform setup work.

Possible architecture:

```text
Your PC
→ point-to-site VPN
→ Azure VNet
→ private VM IP
→ Nginx
→ Docker services
```

Later, add:

```text
Azure Private DNS
private hostname resolution
VPN client DNS configuration
```

This is a separate networking project. It is not required before Docker, Nginx, Terraform, Ansible, or GitHub Actions.

---

# Phase 13 — Kubernetes only after the Docker path is solid

Do not start Kubernetes until you can answer yes to all of these:

```text
[ ] I understand Dockerfiles.
[ ] I can run multiple services with Compose.
[ ] I understand Docker networking.
[ ] I can configure Nginx reverse proxying.
[ ] I can deploy to an Azure VM.
[ ] I can recreate Azure infrastructure using Terraform.
[ ] I can configure a VM using Ansible.
[ ] I have CI checks in GitHub Actions.
[ ] I understand logs, health checks, and recovery.
```

Then Kubernetes becomes a new implementation of concepts you already understand instead of a completely separate world.

---

# Recommended order of the next milestones

```text
1. Finish Flask API health and info endpoints.
2. Replace Traefik with Nginx.
3. Add a static Nginx landing page.
4. Document the request flow and Docker network.
5. Build one or two Python operations scripts.
6. Deploy manually to an Azure VM.
7. Recreate the Azure VM with Terraform.
8. Configure it with Ansible.
9. Add GitHub Actions validation.
10. Add monitoring and log analysis.
```

# Portfolio story

> Built a private-by-default Azure platform lab around a containerised Python API. Used Docker Compose and Nginx to separate public entry points from internal services, deployed the stack to Ubuntu, recreated Azure infrastructure with Terraform, configured the VM using Ansible, and added GitHub Actions validation and operational tooling for logs, health checks, and monitoring.
