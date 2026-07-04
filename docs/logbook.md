
**Entry 1**
The first part of this project was setting up docker with wsl2 ubuntu integration, and then learning how to set up my containerized services.

The first service was from a public image (Pihole), it was set up just by grabbing the compose file from the pihole docker github repo. 
Then i made a .env file with the admin endpoint password and then i was able to set up my first container. 

For the second service i made my own service. A python app using Flask and some other time imports. Just a basic api with two endpoints one being just a welcome to my homelab and the other a time counter for how long the service has been running. Crucially the python code has this piece of code:
                    if __name__ == "__main__":
                        app.run(host="0.0.0.0", port=5000)
This code makes the app listen to requests from all network interfaces.

Then i made a dockerfile for the app that installs Flask from requirements.txt and all that other good stuff like choosing the python image 3.12-slim and making the working directory. 

Then i made the docker-compose for my image which just builds from the dockerfile in the dir, gives a name to the container, and makes the tunnel/opens the ports between windows and the container. And makes the container auto restart unless stopped manually. 

The way i have it setup right now is each servicefolder has its own docker-compose, and then i have a more orchestrating compose at the root. 

Some mistakes ive made so far:
i had the env file in pihole directory, but if we are gonna use the orchestration compose at root we need the env at root.

Obviously i made tons of mistakes making the python app but thats a given since i suck at python. 

Also have had lots of struggles with wsl2 integration with docker and stuff.

**Entry 2**
WSL and docker isnt mixing well, getting many integration errors. Im just gonna spin up a virtual machine in proxmox right now. It was planned for further on but this just isnt working.

**Entry 3**
So: I tried getting an Azure vm up and running but the costs of the available versions were too high, decided on aws instead and got a vm up and running there with these specs:

Provider: AWS EC2
Region: Europe (Stockholm), eu-north-1
Availability Zone: eu-north-1b
Instance name: homelab-ec2-01
Instance type: t3.micro
Compute: 2 vCPUs, 1 GiB RAM
Operating system: Ubuntu Server 26.04 LTS
Architecture: x86_64 / 64-bit x86
Root disk: 8 GiB gp3 EBS volume
Networking: Default VPC and subnet
Inbound access: SSH on port 22 restricted to my current public IP
HTTP/HTTPS: Not exposed
Access method: SSH from Windows Terminal using an EC2 key pair

Will now be installing docker, docker compose and test my services, then it's time to get nginx up and running there.

Wrote this in the terminal to install docker on my server:
# Add Docker's official GPG key:
sudo apt update
sudo apt install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
sudo tee /etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF

sudo apt update

sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

**Entry 4**
Have now managed to get my proxy and my api running with docker compose on my aws ubuntu server! docker ps output here:
ubuntu@ip-172-31-33-11:~/homelab$ docker compose ps
NAME                IMAGE                 COMMAND                  SERVICE       CREATED         STATUS         PORTS
homelab-api         homelab-homelab-api   "python app.py"          homelab-api   8 seconds ago   Up 7 seconds   0.0.0.0:5000->5000/tcp, [::]:5000->5000/tcp
homelab-traefik-1   traefik:v3.7          "/entrypoint.sh --ap…"   traefik       8 seconds ago   Up 7 seconds   0.0.0.0:80->80/tcp, [::]:80->80/tcp, 0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp

What i'm going to do now, is swap traefik for nginx. First ill need to map out my system.
- What is running?
- Which ports are exposed?
- Which container talks to which container?

What's running: my flask api is running, listening on port 5000, traefik is listening on port 80. the traefik dashbord is listening on port 8080. 

Which container talks to which container: the reverse proxy talks to the api

EC2 VM host network
├── port 80
│   └── forwarded into Traefik container
│
├── port 8080
│   └── forwarded into Traefik dashboard
│
└── port 5000
    └── forwarded directly into Flask API container

Docker private network
├── Traefik
│   └── talks to homelab-api:5000
│
└── homelab-api
    └── listens on port 5000 internally

One important thing i have learned today, by dialogue with ChatGPT: 
- The port mappings in the docker compose file dictate on which ports the machine can reach the services, while the service/app configurations themselves dictate on which ports they can be reached "internally". So for example, in my python app; app.run(host="0.0.0.0", port=5000) means that Flask listens inside it's own container on port 5000.
While this code in the compose:
    ports:
        - "5000:5000"
creates the mapping of machine:5000 -> this container:5000

**Entry 4**
Ive now added nginx alongside traefik for now.

The process went like this:
    - Look up the public image, choose a tag, i chose alpine something
    - Make the nginx.config file and set it up i just wrote the server block that makes the container listen on port 80 and forward to the flask api
    - Then make the compose file, where i get the image, open port 8081 on machine to the container 80, add it to the network, and mount the config file into the containers file system. a caveat here was since we only included one block of the config we had to specify a subfolder instead of overwriting the full pre shipped config file. 

Here is the output of docker compose ps: 
    ubuntu@ip-172-31-33-11:~/homelab$ docker compose ps
NAME                IMAGE                 COMMAND                  SERVICE       CREATED          STATUS         PORTS
homelab-api         homelab-homelab-api   "python app.py"          homelab-api   20 seconds ago   Up 9 seconds   0.0.0.0:5000->5000/tcp, [::]:5000->5000/tcp
homelab-nginx-1     nginx:1.30.3-alpine   "/docker-entrypoint.…"   nginx         20 seconds ago   Up 9 seconds   0.0.0.0:8081->80/tcp, [::]:8081->80/tcp
homelab-traefik-1   traefik:v3.7          "/entrypoint.sh --ap…"   traefik       3 hours ago      Up 3 hours     0.0.0.0:80->80/tcp, [::]:80->80/tcp, 0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp

Testing

curl -i http://localhost:8081/health

Returned:

HTTP/1.1 200 OK
Server: nginx/1.30.3

{"status":"healthy"}

This proves the full path works:

curl
→ VM localhost:8081
→ Nginx
→ Flask API /health
→ JSON response back through Nginx

I also tested an unknown path:

curl -i http://localhost:8081/surprise

Nginx still forwards /surprise to Flask. Flask has no route for it, so Flask returns a 404 Not Found, which Nginx passes back to the client.

## Entry #5 — Replaced Traefik with Nginx

I removed Traefik from the active Compose stack and moved Nginx from the temporary test port to the normal HTTP port:

```yaml
ports:
  - "80:80"
```

The VM now receives HTTP traffic on port 80 and Docker forwards it to Nginx on port 80 inside its container.

I also removed the API’s direct port mapping:

```yaml
ports:
  - "5000:5000"
```

The Flask API still listens on port 5000 inside its container, but the EC2 VM no longer exposes that port directly. It can now only be reached through Nginx over the private Docker network.

Final request path:

```text
VM:80
→ Nginx container:80
→ Docker network
→ homelab-api:5000
```

Tests:

```bash
curl http://localhost/
# Welcome to my homelab API!

curl http://localhost/health
# {"status":"healthy"}

curl http://localhost:5000/
# Failed to connect
```

`curl -i http://localhost/health` also returned `Server: nginx/1.30.3`, confirming that Nginx is now the only entry point to the API.

**Entry 6**
I have replaced the old shared 'proxy' docker network with a split edge/backend networks. 

edge has nginx only, while backend has nginx and homelab-api.

nginx is connected to both networks so it acts like a bridge between incoming traffic on port 80 and the api/other services in the future. 

The request path is now:

EC2 host:80
→ Nginx
→ backend network
→ homelab-api:5000

I verified it with docker network inspect:

homelab_edge:
  homelab-nginx-1

homelab_backend:
  homelab-nginx-1
  homelab-api

So the result now is that nginx is the only container that receives traffic from the host and it is the only one communicating directly with the other containers.

**Entry 7**

Today I made my own small TCP service together with a terminal client to test it.

First I made a `README.md` inside the `tcp-service` directory which describes my own application layer protocol. It runs on TCP port 9000, uses UTF-8, and every command ends with `\n`.

The basic flow is:

```text
client opens TCP connection
→ sends one command
→ server responds
→ server closes connection
```

The currently implemented commands are `PING` and `ECHO`.

Examples:

```text
PING
→ PONG

ECHO hello
→ ECHO hello
```

I also made two different error cases:

```text
FAKECOMMAND hello
→ ERROR: unknown command

PING hello
→ ERROR: invalid request

ECHO
→ ERROR: invalid request
```

Then I made `server.py` using Python's built-in socket module. The server binds to `0.0.0.0:9000`, listens for connections, receives data, decodes the bytes from UTF-8, parses the command and sends back a response.

One thing I learned here is that TCP is only a byte stream, so I needed to define my own way of knowing where a command ends. I chose newline terminated commands. I also used `rstrip("\n")` instead of `strip()` so I only remove the protocol newline and not normal spaces from an ECHO request.

After that I made `toolbox/tcp_client.py`.

The client is terminal based. I decided that the client should not validate commands itself. It just takes whatever I type, adds `\n`, sends it to the server and prints the response. This makes it easy to test both valid and invalid commands.

The client opens a new TCP connection for every command because the server closes the connection after responding. `quit` is handled locally by the client and exits the terminal program.

Then I containerized both services.

```text
tcp-service/
├── README.md
├── server.py
├── Dockerfile
└── compose.yaml

toolbox/
├── tcp_client.py
├── Dockerfile
└── compose.yaml
```

Neither service needs a `requirements.txt` because both only use Python standard library modules.

Both services are connected to the internal `backend` Docker network. I did not expose any ports for them, since this TCP service should not be reachable from the internet or through Nginx.

Instead the toolbox connects internally to:

```text
tcp-service:9000
```

This was a good lesson in Docker networking. Inside the toolbox container, `127.0.0.1` means the toolbox container itself, not the TCP service. Docker Compose has internal DNS, so the service name `tcp-service` resolves to the TCP container's internal IP address.

Testing was done with:

```bash
docker compose exec -it toolbox python tcp_client.py
```

The full path now works:

```text
SSH terminal
→ toolbox container
→ Docker internal DNS
→ tcp-service:9000 on backend network
→ response back to terminal
```

**Entry 8**

Today I made a UDP service with a terminal client that synchronizes a live text field between multiple clients.

The client sends `JOIN` when it starts, and the server stores its IP and port as a subscriber. When I type, the client sends `UPDATE <current text>` to the server. The server increases a sequence number and broadcasts `TEXT <sequence> <text>` to every subscribed client.

Unlike the TCP service, UDP has no connection and no newline framing. Each UDP datagram is already one message.

I used `select` in the client so it can listen for both keyboard input and incoming UDP messages at the same time.

I tested it by running two clients in separate terminals. Typing in one updated the other basically instantly. I also used `tcpdump` to see the actual UDP `UPDATE` packets arriving at port 9001 and the server broadcasting the `TEXT` packets back to each client.

**Entry 9**

Today I started moving the AWS VM part of the homelab into Terraform.

The goal for this first Terraform milestone is not to automate everything yet. I just want Terraform to create my AWS network and Ubuntu VM, then I will SSH into it manually. Later Ansible will configure the VM with Docker, Compose, and the services.

I installed Terraform and AWS CLI in WSL and created a separate AWS profile called `homelab-terraform`.

Terraform uses that local profile to authenticate to AWS. The IAM access keys are only for Terraform and AWS CLI to talk to AWS APIs. They are separate from the SSH keys used to log into the Linux VM.

I decided to make my own VPC instead of using the default VPC.

The network so far is:

```text
VPC: 10.0.0.0/16
→ public subnet: 10.0.1.0/24
→ route table
→ Internet Gateway
→ internet
```

The subnet has public IP assignment enabled, and the route table has this route:

```text
0.0.0.0/0
→ Internet Gateway
```

I also associated that route table with the subnet. This means instances launched in the subnet can get a public IP and have a route out to the internet.

Then I created a security group for SSH access.

It allows:

```text
Inbound TCP port 22
→ only from my current public IP /32
```

It also allows all outbound traffic for now.

One thing I learned is that `0.0.0.0/0` means different things depending on where it is used.

Inside the route table it means:

```text
traffic going to any IPv4 destination
→ send it to the Internet Gateway
```

Inside a security group inbound rule it would mean:

```text
allow incoming traffic from any IPv4 address
```

I only allow SSH from my own public IP, so I used `/32`, which represents one specific IPv4 address.

I also made a dedicated SSH key pair for the EC2 VM.

```text
~/.ssh/homelab-ec2
→ private key, stays only on my computer

~/.ssh/homelab-ec2.pub
→ public key
```

I copied only the public key into the repository:

```text
terraform/keys/homelab-ec2.pub
```

Terraform reads that file and registers it in AWS as an EC2 key pair.

The EC2 instance is now explicitly connected to:

```text
Ubuntu AMI
→ custom subnet
→ SSH security group
→ EC2 key pair
```

I also learned that AWS only receives the public half of the SSH key pair. The VM gets the public key, and my local SSH client proves it has the matching private key when I connect. The private key never goes into Terraform, Git, AWS, or the VM.

I accidentally committed Terraform's `.terraform` directory, which included the downloaded AWS provider binary. GitHub rejected the push because the provider binary was around 674 MB.

I fixed that by adding this kind of local Terraform stuff to `.gitignore`:

```text
terraform/.terraform/
terraform/*.tfstate
terraform/*.tfstate.*
terraform/*.tfvars
```

I kept `.terraform.lock.hcl` committed because it locks the provider version, and I kept the public SSH key committed because GitHub Actions will need access to it later.

The final Terraform plan currently says:

```text
Plan: 10 to add, 0 to change, 0 to destroy.
```

The resources Terraform will create are:

```text
VPC
subnet
Internet Gateway
route table
route table association
security group
SSH ingress rule
egress rule
EC2 key pair
EC2 instance
```

The Terraform configuration is ready to apply.

Next step is to confirm my public IP has not changed, run `terraform apply`, get the VM public IP, and SSH into the new Ubuntu VM using the private key.

I ran `terraform apply` and Terraform successfully created all 10 resources.

```text
Apply complete! Resources: 10 added, 0 changed, 0 destroyed.
```

After that I added a Terraform output for the EC2 public IP:

```hcl
output "app_server_public_ip" {
  description = "public ip of the ec2 instance"
  value       = aws_instance.app_server.public_ip
}
```

Running Terraform again made no infrastructure changes, but it printed the public IP of the VM.

I then connected from WSL with:

```bash
ssh -i ~/.ssh/homelab-ec2 ubuntu@<public-ip>
```

The first connection asked me to verify the VM SSH host fingerprint. After accepting it, SSH saved the fingerprint in my local `known_hosts` file.

I initially got:

```text
Permission denied (publickey)
```

I had forgotten that I gave the private key a passphrase when I created it. After entering the correct passphrase, I could SSH into the new Ubuntu VM.

The full SSH path was:

```text
my WSL machine
→ EC2 public IP
→ Internet Gateway
→ public subnet route table
→ EC2 security group allows my IP on TCP 22
→ SSH service on the Ubuntu VM
→ matching public/private SSH key authentication
```

The security group allowed my network connection to reach port 22.

The SSH key pair allowed me to authenticate as the `ubuntu` user. The VM has the public key, while I keep the matching private key locally. The passphrase unlocks that private key locally so SSH can use it.

This completes the first Terraform milestone:

```text
Terraform configuration
→ AWS VPC and public subnet
→ EC2 Ubuntu VM
→ SSH access from my local WSL machine
```

Next step is to use Ansible to configure the VM instead of manually installing Docker and deploying the Compose stack.

**Entry 10**

Today I started the Ansible phase of the homelab.

Terraform now creates the AWS network and Ubuntu VM. The next layer is Ansible, which will configure that VM so I do not have to manually install Docker, clone the repository, or start the Compose stack every time I recreate the server.

The intended separation is:

```text
Terraform
→ creates AWS infrastructure

Ansible
→ configures the Linux VM and deploys the project

Docker Compose
→ defines and runs the containers
```

For the first Ansible milestone, I am deliberately keeping the scope small.

The only goal is to prove this connection path works:

```text
Terraform output
→ Ansible inventory
→ SSH connection to EC2
→ Ansible runs a harmless test
```

I installed Ansible inside my WSL Ubuntu environment.

My WSL machine is the Ansible control node. This means it runs the Ansible commands and connects outward to the EC2 VM.

```text
WSL machine
→ runs Ansible

EC2 Ubuntu VM
→ managed by Ansible over SSH
```

I do not need to install Ansible or an Ansible agent on the EC2 instance. Ansible uses the existing SSH access that I already configured during the Terraform milestone.

Terraform currently outputs the public IP of the EC2 server:

```text
app_server_public_ip = "51.20.95.154"
```

I also already have the SSH key pair used for the VM:

```text
~/.ssh/homelab-ec2
→ private key, stays on my WSL machine

~/.ssh/homelab-ec2.pub
→ public key, registered with AWS and installed on the VM
```

The private key has a passphrase. Instead of putting the path or passphrase into the repository, I will use `ssh-agent` locally. The agent holds the unlocked private key in my current local session so Ansible can reuse it for SSH authentication.

I created an Ansible inventory file:

```text
ansible/inventory.ini
```

The inventory is Ansible’s list of machines it can manage.

My first inventory group is:

```ini
[homelab]
app_server ansible_host=51.20.95.154 ansible_user=ubuntu
```

The structure is:

```ini
[group_name]
host_alias key=value key=value
```

In this case:

```text
homelab
→ a group name I chose

app_server
→ an inventory alias I chose for the EC2 VM

ansible_host
→ the real IP address Ansible should connect to

ansible_user
→ the Linux user Ansible should log in as
```

`app_server` is not the real hostname of the VM. It is just Ansible’s internal name for that managed host.

The inventory can also contain my own variables later, such as:

```text
deploy_path
environment
app_port
```

Those variables do not do anything automatically. They only become meaningful when a future Ansible playbook uses them.

The `ansible_*` variables are different because they are special connection variables that Ansible already understands.

The next step is to load my private key into `ssh-agent` and run Ansible’s ping module against the `homelab` group.

A successful result will be:

```text
pong
```

This is not a normal network ping. It means Ansible successfully connected to the EC2 VM over SSH and was able to run a small check there.

Once that works, the first Ansible connectivity milestone is complete. After that I can start writing playbooks to make the VM into a reproducible Docker host.

I loaded my SSH private key into `ssh-agent` locally and tested the inventory connection with Ansible’s ping module.

```bash
ansible -i inventory.ini homelab -m ping
```

The result was:

```text
app_server | SUCCESS => {
    "ping": "pong"
}
```

This confirmed the first Ansible milestone:

```text
WSL Ansible controller
→ SSH authentication with my existing EC2 key
→ connection to the EC2 VM as ubuntu
→ Ansible module executed successfully on the VM
```

Ansible also printed a warning about discovering Python at `/usr/bin/python3.14`.

This is not an error. Ansible needs Python on the managed host to run most modules. It found the current Python interpreter automatically, and the warning only means that a future Python installation could cause Ansible to discover a different one. For this lab, I left the default behavior unchanged.

After proving connectivity, I created my first playbook:

```text
ansible/playbook.yaml
```

A playbook describes the desired state of the managed host.

The basic hierarchy is:

```text
playbook
→ play
→ tasks
→ Ansible modules
```

My first play targets the `homelab` inventory group and uses:

```yaml
become: true
```

This means Ansible connects as the normal `ubuntu` user, then uses `sudo` when a task needs root permissions.

This is needed for system-level work such as:

```text
installing APT packages
adding package repositories
writing files under /etc
managing Docker through systemd
```

My first task ensured Git was installed:

```text
desired state
→ Git is present on the VM
```

The task reported `ok` rather than `changed`, which showed that Git was already installed in the Ubuntu image.

This was my first practical example of Ansible’s idempotent model.

```text
Ansible does not blindly run installation commands every time.

Instead, it checks:
“Is the desired state already true?”

If yes:
→ ok

If not:
→ changed
```

I then used Ansible to configure Docker from Docker’s official Ubuntu APT repository.

The intended Docker setup sequence became:

```text
ensure prerequisite packages are installed
→ add Docker’s official APT repository
→ refresh APT package metadata
→ install Docker Engine and Compose plugin
→ ensure Docker starts now and after reboot
```

I used the `ansible.builtin.deb822_repository` module for Docker’s repository instead of translating Docker’s manual shell commands literally.

This allowed the repository task itself to manage both:

```text
Docker package source
→ https://download.docker.com/linux/ubuntu

Docker signing key
→ downloaded and managed through the repository definition
```

The repository uses an Ansible fact:

```yaml
suites: "{{ ansible_distribution_release }}"
```

This variable comes from Ansible’s automatic fact gathering step.

For this VM, Ansible discovers the Ubuntu release codename and inserts it into the repository configuration instead of hardcoding a release such as `noble` or `resolute`.

While installing Docker, I hit a useful architecture problem.

At first, I configured the Docker repository for `arm64`. APT could then see Docker ARM packages, but it could not install their dependencies because the actual VM was using the AMD64 package architecture.

The useful checks were:

```bash
dpkg-query -W libc6
apt-config dump
```

They showed:

```text
libc6:amd64
APT::Architecture "amd64";
```

This clarified an important naming detail:

```text
amd64
→ Debian and Ubuntu name for normal 64-bit x86 servers

It does not mean the server necessarily uses an AMD CPU.
It can also be an Intel x86-64 CPU.
```

I corrected the Docker repository architecture to:

```text
amd64
```

After that, Ansible successfully installed:

```text
docker-ce
docker-ce-cli
containerd.io
docker-buildx-plugin
docker-compose-plugin
```

The Compose plugin is important because it provides the modern command:

```bash
docker compose ...
```

rather than the older standalone `docker-compose` command.

I then added a service task to ensure Docker is both running now and automatically starts after a VM reboot.

```text
state: started
→ Docker is running now

enabled: true
→ Docker starts automatically when the VM boots
```

Once Docker was installed, the next task was deploying the repository.

At first I cloned the repository as root under:

```text
/root/homelab
```

This worked, but it was inconvenient because I normally SSH into the VM as the `ubuntu` user.

I changed the Git task so that only this task opts out of the play-level root escalation:

```text
become: false
```

The repository now lives at:

```text
/home/ubuntu/homelab
```

and is owned by the `ubuntu` user.

The Git task is responsible for both first-time cloning and future updates:

```text
first Ansible run
→ clone repository

later Ansible runs
→ fetch and update the checkout from main
```

The deployment path is explicitly set to the repository directory:

```text
/home/ubuntu/homelab
```

This means the server has a predictable location for the Compose project rather than depending on whichever directory Ansible happens to use.

For the Compose deployment itself, I used the `community.docker.docker_compose_v2` module.

The desired state is:

```text
project source
→ /home/ubuntu/homelab

build changed images
→ always

remove services no longer declared in Compose
→ remove_orphans: true

ensure declared services exist and are running
→ state: present
```

I deliberately did not use `docker compose down` before every deployment.

```text
docker compose up -d --build --remove-orphans
```

is a better normal deployment behavior because it rebuilds or recreates only what changed and removes old orphaned containers without intentionally stopping the whole stack first.

After Ansible ran the Compose deployment, I verified the stack from the VM:

```bash
sudo docker compose ps
```

The result showed all current services running:

```text
homelab-api
nginx
tcp-service
toolbox
udp-service
```

Nginx was bound to port 80 on the VM:

```text
0.0.0.0:80->80/tcp
```

I then tested the reverse-proxy path locally on the VM:

```bash
curl -i http://localhost/
```

The response was:

```text
HTTP/1.1 200 OK
Server: nginx/1.30.3
```

This confirms:

```text
localhost:80
→ Nginx container

Nginx
→ successfully serves the configured application response
```

The service is intentionally not publicly reachable yet.

My AWS security group currently allows inbound SSH on port 22 only.

So the current state is:

```text
public internet
→ blocked from port 80 by AWS security group

inside the VM
→ Nginx and the Compose stack are working correctly
```

When I am ready to expose the HTTP service publicly, I should add an inbound TCP port 80 rule through Terraform rather than manually changing AWS settings in the console.

The Ansible deployment milestone is now complete.

```text
Terraform
→ creates the AWS infrastructure

Ansible
→ installs Docker
→ configures Docker’s package repository
→ enables the Docker service
→ clones and updates the repository
→ starts the Compose stack

Docker Compose
→ builds and runs the application containers

Nginx
→ serves the internal HTTP entry point on port 80
```

The resulting deployment flow is now:

```text
Git repository
→ Terraform creates VM
→ Ansible configures VM
→ Ansible deploys repository
→ Docker Compose runs services
→ Nginx serves the application internally
```

The next likely milestones are:

```text
Terraform
→ allow inbound HTTP on port 80 when intentionally ready

GitHub Actions
→ validate Terraform and Ansible changes on pushes

Later deployment automation
→ decide how GitHub Actions should trigger an Ansible deployment
→ eventually build and publish container images through a registry
```

**Entry 11**

Today I moved the main Terraform state for the homelab from my local WSL filesystem into an S3 backend.

Before this, Terraform state only existed locally:

```text
WSL machine
→ terraform.tfstate
```

That worked while I was the only machine running Terraform, but it meant the state was tied to one filesystem.

The goal of this phase was:

```text
local Terraform state
→ private, versioned S3 state
→ same Terraform workflow from WSL
→ ready for future shared use from GitHub Actions
```

I first created a separate Terraform bootstrap root under:

```text
terraform/bootstrap/
```

This exists because Terraform needs the S3 bucket before the main Terraform configuration can use it as its backend.

The bootstrap configuration creates:

```text
S3 bucket
→ versioning enabled
→ Block Public Access enabled
→ normal S3-managed encryption
```

The bootstrap root keeps its own state locally for now.

The main Terraform root does not manage the bucket that stores its own state. Instead, it has a backend configuration that points to the already-created bucket.

The main backend is configured with:

```text
S3 bucket
→ homelab-tfstate-bucket-1337

state object path
→ homelab/main/terraform.tfstate

region
→ eu-north-1

state locking
→ S3 lockfile enabled
```

The state object path is called the S3 key. It is the destination path inside the bucket where Terraform stores the state file.

Before migrating, I copied my local state to a separate backup directory outside the Git repository.

I then ran:

```bash
terraform init -migrate-state
```

Terraform detected that the previous local backend already contained state, while the S3 backend was empty.

It asked whether I wanted to copy the existing state into S3.

After confirming, Terraform migrated the state and initialized the main Terraform root to use S3 going forward.

The flow is now:

```text
Terraform configuration
→ backend.tf points at S3

terraform init
→ reads backend configuration
→ reads and writes state in S3

terraform plan / apply
→ uses the remote S3 state
```

I verified the state object was visible in the S3 bucket through AWS.

I then ran:

```bash
terraform plan
```

Terraform refreshed the existing VPC, subnet, security group, route table, key pair, and EC2 instance from AWS.

The result was:

```text
No changes. Your infrastructure matches the configuration.
```

This confirmed that Terraform successfully read the migrated state from S3 and still matched it to the real AWS infrastructure.

The local `terraform.tfstate` files still exist on my WSL machine, but they are now old local snapshots rather than the active source of truth.

The active state location is now:

```text
S3 bucket
→ homelab-tfstate-bucket-1337

S3 key
→ homelab/main/terraform.tfstate
```

This completes the Terraform remote-state milestone.

The current deployment flow is now:

```text
Git repository
→ Terraform configuration

Terraform bootstrap
→ creates protected S3 state bucket

Terraform main root
→ creates AWS VPC and EC2 infrastructure
→ stores state in S3

Ansible
→ configures the VM
→ deploys the Docker Compose stack
```

The next phase is GitHub Actions validation for Terraform and Ansible changes. Later, GitHub Actions can use this same remote state safely when I decide to automate real Terraform plans and applies.

**Entry 12** Today I made the Terraform bootstrap code for GitHub Actions OIDC authentication to AWS.

The problem is that later GitHub Actions needs to run Terraform against AWS, but I do not want to store a permanent AWS access key as a GitHub secret.

The bootstrap Terraform now creates:

```text
GitHub OIDC provider
→ AWS can verify tokens issued by GitHub Actions

IAM role: homelab_cd_terraform
→ the AWS identity GitHub Actions will assume

IAM policy: homelab-terraform-cd
→ gives the role permissions to manage EC2 resources
→ gives narrow access to the exact S3 Terraform state and lock objects
```

The role trust policy is restricted to:

```text
repo:viper6z/homelab:ref:refs/heads/main
```

This means a workflow running from the `main` branch of this repository can request a short lived GitHub OIDC token. AWS verifies the token, then gives the workflow temporary credentials for `homelab_cd_terraform`.

The resulting future flow will be:

```text
GitHub Actions on main
→ GitHub OIDC token
→ AWS STS verifies token and trust policy
→ temporary credentials for homelab_cd_terraform
→ Terraform can read S3 state and manage EC2 infrastructure
```

I applied the bootstrap stack locally using my existing `homelab-terraform` AWS profile. It created the OIDC provider, IAM role, IAM policy, and policy attachment.

The role ARN is:

```text
arn:aws:iam::127372371185:role/homelab_cd_terraform
```

The next step is still CI first: GitHub Actions should run Terraform formatting and validation on pull requests without AWS credentials. The OIDC role will be used later for the CD workflow on pushes to `main`.

**Entry 12**

Today I made the first CI workflow for the homelab. It runs on pull requests into `main` and only validates the repository. Nothing gets deployed and it does not use AWS credentials.

The workflow has three jobs:

```text
Terraform
→ terraform fmt -check
→ terraform init -backend=false
→ terraform validate

Ansible
→ install ansible-core
→ install the community.docker collection from requirements.yml
→ syntax check the real playbook

Docker Compose
→ docker compose config
```

The Terraform job does not touch the S3 backend because `-backend=false` skips remote state setup. The Ansible job only parses the playbook, so it does not SSH into the VM or run tasks. The Compose job validates the root compose configuration without starting containers.

So the flow is now:

```text
dev branch
→ PR into main
→ CI validates Terraform, Ansible and Compose
→ merge when it is green
```

**Entry 13**

Today I started the CD phase. The workflow now runs only when a pull request into `main` is merged, so direct pushes to `main` do not start a deployment.

The flow so far is:

```text
merged PR
→ GitHub OIDC logs into AWS with temporary credentials
→ Terraform reads the real S3 state
→ terraform plan runs
→ I inspect the plan
→ GitHub production environment waits for my approval
→ terraform apply runs
```

This is working now. The next part is adding Ansible after Terraform apply so the VM also pulls the new repository state and reconciles the Compose stack.

I first thought GitHub Actions could just SSH into the VM, but GitHub-hosted runners have changing public IPs and would need access to my SSH key. Instead I decided to use AWS SSM as the connection path for Ansible.

The CD phase is not finished yet. Next I will add the SSM support resources in the Terraform bootstrap root, then connect Ansible to the EC2 instance through SSM instead of public SSH.

**Entry 14**

Today I started refactoring the deployment flow away from Ansible.

Ansible was useful for understanding configuration management and idempotency, but it does not really justify itself for this project anymore. The VM is just a small Docker host, so the first boot machine setup belongs in cloud-init. Later workload updates will come from GitHub Actions through AWS Systems Manager instead.

I started the SSM IAM setup in the Terraform bootstrap root.

First I created an IAM role called `ssm_role_ec2`. Its trust policy allows the EC2 service to assume the role, and it has AWS's `AmazonSSMManagedInstanceCore` policy attached. This gives the SSM Agent on the VM the permissions it needs to authenticate to Systems Manager.

I then created an instance profile called `ssm_profile` which contains the role. The profile is the thing that will later be attached to the EC2 VM.

I also updated the existing GitHub Actions OIDC role. It can now:

```text
iam:GetInstanceProfile
→ read ssm_profile

iam:PassRole
→ allow the runner to attach ssm_role_ec2 to EC2 only


**Entry 15**

Have attached the SSM profile to the ec2 vm now and confirmed it works, also verified that cloud init worked and set up the docker environment.

Next up is finishing the pipeline with our new ssm plumbing. Now we can do ssm run for git clone, git pull, docker compose up etc. 

Ive added some new permissions to the github_oidc role, allowing it to send commands and read output from the vm through SSM, although ive realized maybe later i should switch to a PR/Prod role split, since the current layout lets github actions run shell command at CI time. 

Tested sending some basic commands through SSM in the cd, they didnt get denied, but didnt reach any host either, but i think its a propagation issue because this was the same ID that the role seal_host got applied to the VM which is the mechanism in which my SSM commands target vms. 

Actually it wasnt a propagation issue rather it was because github actions didnt wait for the time it takes for the response to reach, confirmed in the console that the message succeeded. with the output: 

/var/snap/amazon-ssm-agent/13349

Docker version 29.6.1, build 8900f1d

Apparently, when we send commands through SSM in github actions, we need to capture the CommandId and the InstanceId in order to be able to get information on the outputs and if it succeeded etc.
this could be compared to going up to order at the restaurant where our order is our sendcommand, and our number we get from the cashier to get our food is the commandid and instanceid.

Also added some bash stuff so the workflow doesnt exit if the command doesnt execute rather it waits until we have queried the command invocation for info 

After that I replaced the test commands with the actual first deployment logic.

The VM now checks if `/opt/seal/.git` exists.

If it does not exist, it clones the repository directly into `/opt/seal`.

If it already exists, it runs `git pull --ff-only` so it only updates if the VM checkout has not been manually changed or diverged.

After either path it runs:

```text
docker compose up -d --build --remove-orphans

I added set -e so the deployment stops immediately if Git or Docker Compose fails instead of continuing and pretending the deploy worked.

The first real deploy succeeded. GitHub Actions found the VM, sent the SSM command, the VM cloned the repository into /opt/seal, Docker pulled and built the images, and Compose started the stack.

GitHub Actions
→ SSM
→ git clone / git pull
→ docker compose up
→ running workload

This means the Ansible deployment layer is now effectively gone.

Cloud-init handles the first boot Docker host setup, while SSM handles actual workload deployments on the existing VM.

**Entry 16**

Today I finished testing the normal CI/CD update path, not just the first deploy clone path.

The goal of this pipeline is that `main` is the intended state of the system. Instead of SSHing into the VM manually to pull changes and rebuild the Compose stack, a merged PR should make AWS converge toward what is in the repository.

I made a small application-only change to the `/health` endpoint:

```json
{"ci/cd":"tested","status":"healthy"}

The PR went through CI first, which validates the Terraform and Docker Compose layers. After merge, CD ran Terraform plan, waited for production approval, applied Terraform, found the VM through the Role=seal-host tag, and sent the deployment command through SSM.

This time /opt/seal already contained the repository, so the VM used:

git pull --ff-only
→ docker compose up -d --build --remove-orphans

I then verified from the VM that all Compose services were up and that Nginx returned:

HTTP/1.1 200 OK
{"ci/cd":"tested","status":"healthy"}

This proves the normal later deployment path works:

PR
→ CI validation
→ merge to main
→ GitHub Actions OIDC
→ Terraform apply
→ SSM
→ git pull
→ Docker Compose rebuild
→ updated running workload

For now, a bad deployment can be reverted through Git by merging a revert PR, which lets the same pipeline reconcile the VM back toward the earlier working code.

**Entry 17** 
Today im starting work on nginx and adding a pulic facing http ingress to my lab. This of course requires hardening the security and what is reachable.

Im starting with reconfiguring my nginx.conf file. sort of like an acl type of mindset where i explicitly write out the exact requests that will be valid and where to point them and then i have a catchall at the bottom that returns 404. 

The old config had a broad location / block which proxied every request path to the API. That meant any current or future Flask route would become public once port 80 was opened.

I changed this to exact Nginx locations for the harmless routes I intend to expose: /, /health, and /time. Anything else now hits the catchall and returns a 404 directly from Nginx without ever reaching Flask. This also makes it so that any new api route that's created is by default not reachable publically.

I had to make nginx reload its config in the CD pipeline. So i added the command for that but it still didnt work. The reload command reloads the configuration file that is runnign INSIDE the container i.e the old one. So my changes to the file, even if they reach the VM doesnt reach the container. So i need to figure out a way to get the changes into the running container.

I could use docker cp, which supports copying a host file into the running container. But i want a cleaner way that just makes nginx use the new file instead of updating its stale file.

What i am going to do is instead of mounting a specific file im going to mount the config directory. So that the path will resolve to the current one very time.

Put the nginx config in its own sub directory and mounted that in the compose file and now it works.

**Entry 18**
Now its time to work on the public ingress through AWS to the machine. We want to change the inbound rules on the security group attached to the ec2 instance.

THis is the plan: 
Current:
security group called “allow SSH”
→ port 22 from your IP
→ all outbound

New:
security group representing the Seal host
→ port 80 from anywhere
→ no inbound SSH
→ all outbound stays

It seems like changing the name and stuff of a security group can be quite complicated with how terraform handles it. These are apparently immutable properties so changing this will lead to terraform having to replace the whole security group, not the end of the world though the only headache is you need to make sure you update the resources that attach to the security group so they reattach to the new one.

The replacement is hanging. I got this error:
│ Error: deleting Security Group (sg-06969b19f147c7522): operation error EC2: DeleteSecurityGroup, https response error StatusCode: 400, RequestID: aed36be9-0728-4aba-a19b-45213abf7259, api error DependencyViolation: resource sg-06969b19f147c7522 has a dependent object

im going to add the lifecycle create before destroy = true so that we make the new security group before destroying the old one. then we can move the VM before deleting the old SG because we cant delete it as long as it has dependent resources.

With the new security group, i can reach my VM through my browser with http! and i can go through nginx to reach my exposed api endpoints!
http://13.51.193.197/health returns the json response
http://13.51.193.197/random (not a real endpoint/exposed) returns nginx 404, so it doesnt even reach the API. 

With this i have officially touched on all the points in the roadmap.sh beginner devops roadmap and i will now be pausing this lab and do a separate kubernetes lab.
