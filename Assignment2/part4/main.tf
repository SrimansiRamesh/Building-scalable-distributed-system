# You probably want to keep your ip address a secret as well
variable "ssh_cidr" {
  type        = string
  description = "Your home IP in CIDR notation"
}

# name of the existing AWS key pair
variable "ssh_key_name" {
  type        = string
  description = "Name of your existing AWS key pair"
}

variable "ami_id" {
  type        = string
  description = "AMI ID for Amazon Linux 2023"
  default     = "ami-0532be01f26a3de55"  
}

# The provider of your cloud service, in this case it is AWS. 
provider "aws" {
  region     = "us-east-1" # Which region you are working on
}

# Your ec2 instance
resource "aws_instance" "demo-instance-1" {
  ami                    = var.ami_id
  instance_type          = "t2.micro"
  iam_instance_profile   = "LabInstanceProfile"
  vpc_security_group_ids = [aws_security_group.ssh.id]
  key_name               = var.ssh_key_name

  tags = {
    Name = "terraform-instance-1"
  }
}

resource "aws_instance" "demo-instance-2" {
  ami                    = var.ami_id
  instance_type          = "t2.micro"
  iam_instance_profile   = "LabInstanceProfile"
  vpc_security_group_ids = [aws_security_group.ssh.id]
  key_name               = var.ssh_key_name

  tags = {
    Name = "terraform-instance-2"
  }
}

# Your security that grants ssh access from 
# your ip address to your ec2 instance
resource "aws_security_group" "ssh" {
  name        = "allow_ssh_from_me"
  description = "SSH from a single IP"
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_cidr]
  }
  ingress {
    description = "Go API"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [var.ssh_cidr]   
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

output "ec2_public_dns_1" {
  value = aws_instance.demo-instance-1.public_dns
}

output "ec2_public_ip_1" {
  value = aws_instance.demo-instance-1.public_ip
}

output "ec2_public_dns_2" {
  value = aws_instance.demo-instance-2.public_dns
}

output "ec2_public_ip_2" {
  value = aws_instance.demo-instance-2.public_ip
}