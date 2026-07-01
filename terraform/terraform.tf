
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

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-resolute-26.04-amd64-server-*"]
  }

  owners = ["099720109477"] # Canonical
}

resource "aws_instance" "app_server" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.main.id
  vpc_security_group_ids = [aws_security_group.allow_ssh.id]
  key_name               = aws_key_pair.homelab.key_name

  user_data                   = file("${path.module}/cloud-init.yaml")
  user_data_replace_on_change = true
  iam_instance_profile        = data.aws_iam_instance_profile.ec2_ssm.name #instance profile for ssm
  tags = {
    Name = "oskar-terraform-server"
  }
}

resource "aws_key_pair" "homelab" {
  key_name   = "homelab-ec2"
  public_key = file("${path.module}/keys/homelab-ec2.pub")
}

data "aws_iam_instance_profile" "ec2_ssm" { #the instance profile we created in bootstrap
  name = "ssm_profile"
}


