A. Host bootstrap only
Install Docker, enable it, create directories/users, verify the machine is ready. You still deploy Compose manually at first.

Smallest learning step
Clear Terraform/Ansible boundary
Does not yet give one-command app deployment

B. Host bootstrap + Compose deployment
Ansible configures Docker, gets the repo or files onto the VM, creates needed configuration, and runs Compose.

Produces the first genuinely reproducible lab deployment
Introduces app-deployment decisions immediately

C. Full deployment automation from CI
GitHub Actions runs Terraform/Ansible after a merge.

Valuable eventual milestone
Adds secrets, SSH access, runner permissions, and deployment safety too early unless the manual path works first









































































































