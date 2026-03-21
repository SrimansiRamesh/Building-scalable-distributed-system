variable "service_name" {
  type        = string
  description = "Base name for RDS resources"
}

variable "subnet_ids" {
  type        = list(string)
  description = "Subnets for the RDS DB subnet group"
}

variable "ecs_security_group_id" {
  type        = string
  description = "Security group ID attached to ECS tasks (source for DB ingress)"
}

variable "db_username" {
  type        = string
  description = "Master username for the MySQL instance"
  default     = "cs6650admin"
}

variable "db_name" {
  type        = string
  description = "Initial database name to create"
  default     = "app"
}

variable "engine_version" {
  type        = string
  description = "MySQL engine version to deploy"
  default     = "8.0.45"
}

variable "db_instance_class" {
  type        = string
  description = "RDS instance class"
  default     = "db.t3.micro"
}

variable "allocated_storage" {
  type        = number
  description = "Allocated storage (GiB)"
  default     = 20
}

variable "backup_retention_period" {
  type        = number
  description = "Backup retention period in days"
  default     = 0
}

variable "apply_immediately" {
  type        = bool
  description = "Whether to apply changes immediately"
  default     = true
}

variable "skip_final_snapshot" {
  type        = bool
  description = "Skip final snapshot on destroy"
  default     = true
}

variable "deletion_protection" {
  type        = bool
  description = "Enable/disable deletion protection"
  default     = false
}

