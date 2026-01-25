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
  default     = "ami-055a9df0c8c9f681c"  
}

# The provider of your cloud service, in this case it is AWS. 
provider "aws" {
  region     = "us-west-2" # Which region you are working on
}

# Your ec2 instance
resource "aws_instance" "demo-instance" {
  ami                    = var.ami_id
  instance_type          = "t2.micro"
  iam_instance_profile   = "LabInstanceProfile"
  vpc_security_group_ids = [aws_security_group.ssh.id]
  key_name               = var.ssh_key_name

  tags = {
    Name = "terraform-created-instance-:)"
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

output "ec2_public_dns" {
  value = aws_instance.demo-instance.public_dns
}

output "ec2_public_ip" {
  value = aws_instance.demo-instance.public_ip
}