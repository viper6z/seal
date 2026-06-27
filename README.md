# homelab

I recently graduated with a B.Sc. in Computer Engineering and started this homelab to build practical cloud and platform engineering skills from the ground up: Linux, containers, networking, infrastructure as code, configuration management, and deployment automation.

## Current architecture

```text
WSL Ubuntu on my laptop
│
├── Terraform
│   └── creates AWS networking and the EC2 VM
│
└── Ansible
    └── connects to the VM over SSH as ubuntu
        └── installs Docker and Git, gets the repository, and runs Compose

AWS
└── EC2 Ubuntu VM
    └── Docker Compose
        ├── nginx
        ├── homelab-api
        ├── tcp-service
        ├── udp-service
        └── toolbox
```

The HTTP request path inside the VM is:

```text
EC2 host:80
→ nginx container:80
→ backend Docker network
→ homelab-api:5000
```

Nginx is attached to both Docker networks. It is the only service on `edge`, and it can reach the application services through `backend`.

```text
edge
└── nginx

backend
├── nginx
├── homelab-api
├── tcp-service
├── udp-service
└── toolbox
```

The EC2 security group currently allows inbound SSH on port 22 only from my own public IP. Nginx is running and bound to port 80 on the VM, but port 80 is intentionally not open to the public internet yet.

## What is currently running

| Component     | What it does                                                                            |
| ------------- | --------------------------------------------------------------------------------------- |
| `homelab-api` | Small Flask API with welcome, uptime, and health endpoints.                             |
| `nginx`       | The only host-facing HTTP entry point. It proxies requests to `homelab-api:5000`.       |
| `tcp-service` | Small custom TCP application protocol on port 9000.                                     |
| `udp-service` | Small UDP live-text synchronization service on port 9001.                               |
| `toolbox`     | Interactive terminal client container used to test the TCP and UDP services internally. |
| `terraform/`  | Creates the AWS network, security group, EC2 key pair, and Ubuntu EC2 instance.         |
| `ansible/`    | Configures the EC2 VM and deploys the Compose project.                                  |

## Services

### HTTP API

`homelab-api` is a small Flask service built from its own Dockerfile. It listens on `0.0.0.0:5000` inside its container so other containers on the backend network can reach it.

Endpoints:

```text
GET /
→ Welcome to my homelab API!

GET /time
→ JSON uptime response

GET /health
→ {"status":"healthy"}
```

The API does not publish a host port. It is reachable through Nginx only.

### Nginx

Nginx listens on port 80 and proxies requests to the API through Docker Compose DNS:

```text
http://homelab-api:5000
```

This means the API is addressed by its Compose service name, not by a hardcoded container IP.

### TCP service

The TCP service is a small custom application-layer protocol built using Python's standard `socket` module.

```text
Transport: TCP
Port: 9000
Encoding: UTF-8
Framing: one command per line, terminated by \n
```

The connection flow is deliberately simple:

```text
client opens a connection
→ sends one command
→ server responds
→ server closes the connection
```

Current commands:

```text
PING
→ PONG

ECHO hello
→ ECHO hello
```

The service also returns explicit errors for unknown commands and invalid requests. It is only available inside the `backend` Docker network.

### UDP live-text service

The UDP service is a small real-time text relay. Clients join the server, send their current text, and receive the latest text from the server.

```text
Client start
→ JOIN

Client input changes
→ UPDATE <current text>

Server broadcast
→ TEXT <sequence> <text>
```

The server increases a sequence number for each update. Clients ignore older `TEXT` messages, which gives a basic way to deal with UDP messages arriving out of order.

This is intentionally not a collaborative editor. UDP does not guarantee delivery or ordering, and if multiple clients type at the same time, the newest server update wins. The next update contains the full text, so missing one update is acceptable for this small experiment.

## AWS infrastructure

Terraform currently creates:

```text
VPC: 10.0.0.0/16
→ public subnet: 10.0.1.0/24
→ Internet Gateway
→ route table with 0.0.0.0/0 through the Internet Gateway
→ security group
→ EC2 key pair
→ Ubuntu EC2 instance
```

The VM is an amd64 EC2 host in AWS Stockholm (`eu-north-1`). It has a public IP so I can connect from WSL over SSH, but inbound access is restricted to SSH from my own IP.

Terraform state is currently local to my WSL environment. Moving it to an S3 backend is planned before Terraform is allowed to run real plans or applies from GitHub Actions.

## Ansible deployment

My WSL machine is the Ansible control node. It connects to the EC2 VM as the `ubuntu` user using the same SSH key pair that Terraform registers with AWS.

The playbook currently does this:

```text
ensure Git is installed
→ add Docker's official APT repository
→ install Docker Engine, Buildx, and the Compose plugin
→ start Docker and enable it at boot
→ clone or update this repository on the VM
→ run the Compose project with community.docker.docker_compose_v2
```

The goal is that a newly created VM can be turned into the running homelab without manually reinstalling Docker, cloning the repository, or starting containers by hand.

## Running the Compose stack

The root `compose.yaml` includes the service Compose files, so the normal local command is run from the repository root:

```bash
docker compose up --build
```

`docker compose up` creates and starts the declared services. `--build` tells Compose to build the local service images before starting the stack.

To see the running containers:

```bash
docker compose ps
```

`ps` prints the services in this Compose project, their status, and any host ports they publish.

To test the Nginx to API path from the machine running Compose:

```bash
curl -i http://localhost/
curl -i http://localhost/health
```

`curl` makes an HTTP request. `-i` includes the response headers, which makes it easy to confirm that Nginx handled the response.

To stop and remove the current Compose containers and networks:

```bash
docker compose down
```

`down` stops the Compose services and removes the project resources it created. It does not delete locally built images unless extra flags are added.

## Testing the protocol services

The toolbox container is attached to the internal `backend` network and has an interactive terminal enabled.

Start the TCP client:

```bash
docker compose exec -it toolbox python tcp_client.py
```

`exec` runs a command inside an already running service container. `-i` keeps standard input open and `-t` allocates a terminal, which makes the client usable interactively.

Start the UDP client:

```bash
docker compose exec -it toolbox python udp_client.py
```

Run the UDP client in two separate terminals to see one client update the other. The service directory also contains a recorded demo in `docs/udp-live-demo.gif`.

## Terraform and Ansible workflow

Terraform is used for infrastructure. Ansible is used for configuring the Linux VM and deploying the Compose stack.

```text
Terraform
→ creates AWS infrastructure

Ansible
→ configures the VM and deploys the repository

Docker Compose
→ defines and runs the containers
```

From the `terraform` directory, the normal Terraform flow is:

```bash
terraform init
terraform plan
terraform apply
```

* `terraform init` downloads the provider plugins and initializes the working directory.
* `terraform plan` compares the Terraform configuration with state and shows the changes it would make.
* `terraform apply` performs the changes after showing the plan and asking for confirmation.

From the `ansible` directory, the connection can be tested with:

```bash
ansible -i inventory.ini homelab -m ping
```

* `-i inventory.ini` tells Ansible which inventory file to use.
* `homelab` selects the inventory group.
* `-m ping` runs Ansible's ping module. This is an SSH and module execution check, not an ICMP network ping.

Then the playbook can be run with:

```bash
ansible-playbook -i inventory.ini playbook.yaml
```

`ansible-playbook` applies the desired state defined in `playbook.yaml` to the hosts from the inventory.

## Repository layout

```text
.
├── ansible/            # EC2 inventory and deployment playbook
├── docs/               # Logbook, plans, TODOs, and protocol demo
├── homelab-api/        # Flask API and Dockerfile
├── nginx/              # Nginx Compose service and reverse-proxy config
├── tcp-service/        # Custom TCP protocol server and protocol README
├── terraform/          # AWS infrastructure as code
├── toolbox/            # Internal interactive TCP and UDP clients
├── udp-service/        # UDP live-text synchronization server
└── compose.yaml        # Root Compose file that includes the services
```

## Things I have learned so far

* A container port and a host port are different things. The application listens inside its own container, while Docker port publishing decides whether the host can reach it.
* `127.0.0.1` inside a container means that container itself, not another container or the EC2 host.
* Docker Compose gives services internal DNS. `tcp-service:9000` and `homelab-api:5000` work because the service name resolves on the shared Docker network.
* TCP is a byte stream, so an application protocol needs its own framing. The TCP service uses newline-terminated commands.
* UDP datagrams are already message boundaries, but delivery and ordering are not guaranteed. The UDP experiment uses full-state updates and sequence numbers to make that visible.
* A VPC route for `0.0.0.0/0` means traffic can leave through the Internet Gateway. An inbound security-group rule for `0.0.0.0/0` would mean anyone on the internet can reach that port. Those are very different uses of the same CIDR.
* Terraform and Ansible solve different layers of the same deployment path. Terraform creates AWS resources. Ansible configures the operating system and runs the application stack.

## What is deliberately not implemented yet

This is a learning project, so I am adding layers when they have a clear purpose. These are not in place yet:

* public HTTP access through the AWS security group
* TLS and a public domain
* GitHub Actions CI/CD
* remote Terraform state in S3
* a container registry and pre-built deployment images
* Kubernetes

## Next steps

The next phase is CI/CD.

The first goal is a safe CI workflow that validates the repository without touching AWS or the live EC2 VM:

```text
Terraform formatting and validation
→ Ansible syntax validation
→ Compose configuration validation
→ Docker image builds
→ Python checks and tests as the services grow
```

After that, the plan is to make a normal application change follow this path:

```text
change Python, Nginx, Compose, or Ansible
→ CI passes
→ GitHub Actions triggers Ansible
→ Ansible updates the EC2 VM
→ Docker Compose rebuilds or recreates the changed services
```

Terraform infrastructure deployment will stay a separate path. Before GitHub Actions runs real Terraform plans or applies, the local Terraform state will move to an S3 backend so both WSL and GitHub Actions use the same source of truth.

## Notes

The detailed build-up of this project is in [docs/logbook.md](docs/logbook.md). The TCP and UDP service directories also have their own protocol-focused READMEs.

This repository is intentionally evolving as I learn. The current system is small, but it is already one integrated path:

```text
Git repository
→ Terraform creates AWS infrastructure
→ Ansible configures the VM
→ Docker Compose runs the services
→ Nginx provides the HTTP entry point
```
