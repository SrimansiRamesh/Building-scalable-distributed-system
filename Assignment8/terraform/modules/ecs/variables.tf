variable "service_name" {
  type        = string
  description = "Base name for ECS resources"
}

variable "image" {
  type        = string
  description = "ECR image URI (with tag)"
}

variable "container_port" {
  type        = number
  description = "Port your app listens on"
}

variable "subnet_ids" {
  type        = list(string)
  description = "Subnets for FARGATE tasks"
}

variable "security_group_ids" {
  type        = list(string)
  description = "SGs for FARGATE tasks"
}

variable "execution_role_arn" {
  type        = string
  description = "ECS Task Execution Role ARN"
}

variable "task_role_arn" {
  type        = string
  description = "IAM Role ARN for app permissions"
}

variable "log_group_name" {
  type        = string
  description = "CloudWatch log group name"
}

variable "ecs_count" {
  type        = number
  default     = 1
  description = "Desired Fargate task count"
}

variable "region" {
  type        = string
  description = "AWS region (for awslogs driver)"
}

variable "cpu" {
  type        = string
  default     = "256"
  description = "vCPU units"
}

variable "memory" {
  type        = string
  default     = "512"
  description = "Memory (MiB)"
}

# RDS / MySQL — optional; when set, passed into the task as DB_* env vars
variable "db_host" {
  type        = string
  description = "MySQL host (RDS endpoint)"
  default     = ""
}

variable "db_port" {
  type        = string
  description = "MySQL port"
  default     = "3306"
}

variable "db_name" {
  type        = string
  description = "MySQL database name"
  default     = ""
}

variable "db_user" {
  type        = string
  description = "MySQL user"
  default     = ""
}

variable "db_password" {
  type        = string
  description = "MySQL password"
  sensitive   = true
  default     = ""
}

variable "dynamodb_table_name" {
  type        = string
  description = "DynamoDB table name for shopping carts"
  default     = ""
}

variable "storage_backend" {
  type        = string
  description = "Storage backend to use: 'mysql' or 'dynamodb'"
  default     = ""
}
