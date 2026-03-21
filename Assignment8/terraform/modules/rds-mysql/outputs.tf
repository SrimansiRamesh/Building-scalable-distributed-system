output "db_instance_id" {
  description = "RDS DB instance identifier"
  value       = aws_db_instance.this.id
}

output "db_endpoint" {
  description = "RDS endpoint address"
  value       = aws_db_instance.this.address
}

output "db_port" {
  description = "RDS port"
  value       = aws_db_instance.this.port
}

output "db_name" {
  description = "Database name"
  value       = aws_db_instance.this.db_name
}

output "db_username" {
  description = "Master username"
  value       = var.db_username
}

output "db_password" {
  description = "Master password (sensitive)"
  value       = random_password.db_password.result
  sensitive   = true
}

