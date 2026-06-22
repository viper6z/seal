
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

