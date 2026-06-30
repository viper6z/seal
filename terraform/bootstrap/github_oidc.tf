#oidc creation

#trust github to issue tokens
module "github-oidc-provider" {
  source  = "terraform-module/github-oidc-provider/aws"
  version = "2.2.2"

  create_oidc_provider = true
  create_oidc_role     = true

  role_name = "homelab_cd_terraform"

  repositories = [
    "viper6z/homelab:pull_request",
    "viper6z/homelab:environment:production",
  ]

  oidc_role_attach_policies = [
    aws_iam_policy.cd_identity.arn
  ]
}

data "aws_iam_policy_document" "cd_identity" {
  statement {
    actions   = ["ec2:*"]
    resources = ["*"]
  }

  statement {
    actions   = ["s3:ListBucket"]
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

  statement {
    actions = [
      "iam:PassRole"
    ]
    resources = [aws_iam_role.ssm_role.arn]

    
    condition {
      test     = "StringEquals"
      variable = "iam:PassedToService"
      values   = ["ec2.amazonaws.com"]
    }

  }

  statement {
    actions = [
      "iam:GetInstanceProfile"
    ]
    resources = [aws_iam_instance_profile.ec2_ssm_profile.arn]
  }
}

resource "aws_iam_policy" "cd_identity" {
  name   = "homelab-terraform-cd"
  policy = data.aws_iam_policy_document.cd_identity.json
}

