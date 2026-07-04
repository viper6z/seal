
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.92"
    }
  }

  required_version = ">= 1.2"
}

provider "aws" {
  region = "eu-north-1"
}


resource "aws_instance" "app_server" {
  ami                    = "ami-0c851798b239aa71a"
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.main.id
  vpc_security_group_ids = [aws_security_group.seal_host.id]
  key_name               = aws_key_pair.homelab.key_name

  user_data                   = file("${path.module}/cloud-init.yaml")
  user_data_replace_on_change = true
  iam_instance_profile        = data.aws_iam_instance_profile.ec2_ssm.name #instance profile for ssm
  tags = {
    Name = "oskar-terraform-server"
    Role = "seal-host"
  }
}

resource "aws_key_pair" "homelab" {
  key_name   = "homelab-ec2"
  public_key = file("${path.module}/keys/homelab-ec2.pub")
}

data "aws_iam_instance_profile" "ec2_ssm" { #the instance profile we created in bootstrap
  name = "ssm_profile"
}


