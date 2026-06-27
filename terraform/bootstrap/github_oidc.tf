#oidc creation

#trust github to issue tokens
module "github-oidc-provider" {
  source  = "terraform-module/github-oidc-provider/aws"
  version = "2.2.2"
  create_oidc_provider = true
  create_oidc_role     = false
}

data "aws_iam_policy_document" "plan_identity" {
  statement {
    actions = ["ec2:Describe*"]
    resources = ["*"]
  }

  statement {
    actions = ["s3:ListBucket"]
    resources = ["arn:aws:s3:::homelab-tfstate-bucket-1337"]
  }

  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
    ]
    resources = ["arn:aws:s3:::homelab-tfstate-bucket-1337/homelab/main/terraform.tfstate"]
  }

  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["arn:aws:s3:::homelab-tfstate-bucket-1337/homelab/main/terraform.tfstate.tflock"]
  }
}

