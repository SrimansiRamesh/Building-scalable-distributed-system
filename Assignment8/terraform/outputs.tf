output "ecs_cluster_name" {
  description = "Name of the created ECS cluster"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "Name of the running ECS service"
  value       = module.ecs.service_name
}

output "rds_mysql_endpoint" {
  description = "RDS MySQL endpoint address"
  value       = module.rds_mysql.db_endpoint
}

output "rds_mysql_port" {
  description = "RDS MySQL port"
  value       = module.rds_mysql.db_port
}

output "rds_mysql_db_name" {
  description = "RDS MySQL database name"
  value       = module.rds_mysql.db_name
}

output "rds_mysql_username" {
  description = "RDS MySQL master username"
  value       = module.rds_mysql.db_username
}

output "rds_mysql_password" {
  description = "RDS MySQL master password (sensitive)"
  value       = module.rds_mysql.db_password
  sensitive   = true
}

output "dynamodb_table_name" {
  description = "DynamoDB shopping carts table name"
  value       = module.dynamodb.table_name
}

output "dynamodb_table_arn" {
  description = "DynamoDB shopping carts table ARN"
  value       = module.dynamodb.table_arn
}