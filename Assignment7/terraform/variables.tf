variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "service_name" {
  type    = string
  default = "order-api"
}

variable "ecr_repository_name" {
  type    = string
  default = "order-api"
}

variable "container_port" {
  type    = number
  default = 8080
}

variable "ecs_count" {
  type    = number
  default = 1
}

variable "log_retention_days" {
  type    = number
  default = 7
}

variable "worker_count" {
  type        = number
  default     = 1
  description = "Number of concurrent payment-processing goroutines in the processor task"
}
