[ ] Create bootstrap Terraform for S3 state bucket
[ ] Migrate current local state to S3
[ ] Verify local terraform plan reports no unexpected changes

[ ] Create one broad CI workflow
    - Terraform validation
    - Ansible validation
    - Compose validation/build
    - Python checks

[ ] Choose and build app deployment transport
    - GitHub Actions → Ansible → EC2
    - deploy Python/Compose changes automatically

[ ] Add Terraform plan/apply workflow
    - remote state
    - AWS authentication
    - approval before apply

[ ] Later: Terraform-created VM automatically triggers Ansible bootstrap







































































































