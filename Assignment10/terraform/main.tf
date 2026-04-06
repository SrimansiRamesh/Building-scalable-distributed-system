terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

# ECR repository — single image deployed 5 times with different env vars
resource "aws_ecr_repository" "kv_node" {
  name                 = "a10-kv-node"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }
}

output "ecr_repository_url" {
  value = aws_ecr_repository.kv_node.repository_url
}

# Reuse LabRole from Learner Lab (same as A7/A8)
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}