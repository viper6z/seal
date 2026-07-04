#my custom VPC
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "homelab-vpc-1"
  }
}

resource "aws_subnet" "main" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "eu-north-1a"
  map_public_ip_on_launch = true

  tags = {
    Name = "homelab-subnet-1"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "homelab-gw-1"
  }
}

resource "aws_route_table" "rt" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }

  tags = {
    Name = "homelab-rt-1"
  }
}

resource "aws_route_table_association" "rta" {
  subnet_id      = aws_subnet.main.id
  route_table_id = aws_route_table.rt.id
}

moved {
  from = aws_security_group.allow_ssh
  to   = aws_security_group.seal_host
}

resource "aws_security_group" "seal_host" {
  name        = "seal_host"
  description = "Allow incoming tcp on port 80 and all outbound"
  vpc_id      = aws_vpc.main.id

  tags = {
    Name = "homelab-seal-host"
  }
}

resource "aws_vpc_security_group_ingress_rule" "allow_tcp_80" {
  security_group_id = aws_security_group.seal_host.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 80
  ip_protocol       = "tcp"
  to_port           = 80
}

resource "aws_vpc_security_group_egress_rule" "allow_all_traffic_ipv4" {
  security_group_id = aws_security_group.seal_host.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1" # semantically equivalent to all ports
}

