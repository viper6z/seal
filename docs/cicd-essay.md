Why does my homelab need ci/cd and how did i implement it?
Without ci/cd, there is more manual labour. I would have to SSH into my machine and pull the changes and rebuild the compose environment when i change my services. When i change terraform code i would have to run it locally, and then i would have to manually make sure the runtime is still working. Additionally, without ci/cd, there is a scenario where my code on the main branch differs from what is actually real on my machine i.e config drift.

With ci/cd, every PR gets validated; only a merge to main triggers CD. 

CI in this repo is a series of lints/tests. It validates the terraform layer aswell as the docker compose layer in terms of syntax and formatting etc. There are no application tests for the moment.

CD in this repo runs terraform plan, then operator approval into the production environment where it runs terraform apply (with cloud init on first boot that installs git, docker, and docker compose), then it runs a bash script that checks if the working dir contains the repo and cloens the repo if it doesnt exist already, then the standard procedure is git pull --ff-only into docker compose up -d --build --remove-orphans

So from a merge to running containers this happens: 
The CI runs and the runner checks out the repo and validates the terraform code and the compose config.
Then after merge, CD runs.
The github actions runner connects to AWS via OIDC. GitHub Actions requests an OIDC token from GitHub. AWS trusts GitHub’s OIDC issuer and the claims allowed by the role trust policy, so the runner can assume the CD role and receive temporary AWS credentials. It runs terraform plan. Then i approve the plan, and the deploy job begins. the runner starts by applying the terraform plan. When the terraform apply is done, the step of sending deploy commands through SSM commences: what we do here is we get the instance ID of the machine with a ec2 describe-instances query filtered by our Role=seal-host tag. The runner sends an SSM command to that instance. The remote script checks whether /opt/seal/.git exists. On the first deployment it clones the repository directly into /opt/seal. On later deployments it runs git pull --ff-only. It then runs docker compose up -d --build --remove-orphans from /opt/seal. we then save the command ID of this ssm command in order to listen for its result. important thing here is we use set +e around the SSM waiter so the GitHub runner does not exit immediately if the remote command fails or does not reach Success. This lets the workflow still query get-command-invocation and print the remote status, output, and error details before failing the job with the saved waiter exit code.

The overarching roles of the different technologies in the stack is as follows:

terraform is what creates and manages the aws infrastructure

cloud-init is what makes a brand new vm into a usable docker host 

SSM is what enables remote commands

docker compose is what orchestrates and reconciles the workload containers

github actions orchestrates all of this

The goal of this pipeline is that main is the intended state of the system.
When CD succeeds, the AWS infrastructure and running Compose workload have
converged toward what is declared in the repository. If CD fails, the
repository is still the desired state, but the runtime needs investigation
or a revert commit.
