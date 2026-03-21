# Wire together four focused modules: network, ecr, logging, ecs.

module "network" {
  source         = "./modules/network"
  service_name   = var.service_name
  container_port = var.container_port
}

module "ecr" {
  source          = "./modules/ecr"
  repository_name = var.ecr_repository_name
}

module "logging" {
  source            = "./modules/logging"
  service_name      = var.service_name
  retention_in_days = var.log_retention_days
}

# Reuse an existing IAM role for ECS tasks
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

module "rds_mysql" {
  source                = "./modules/rds-mysql"
  service_name          = var.service_name
  subnet_ids            = module.network.subnet_ids
  ecs_security_group_id = module.network.security_group_id
}

module "dynamodb" {
  source     = "./modules/dynamodb"
  table_name = "shopping-carts"
}

module "ecs" {
  source             = "./modules/ecs"
  service_name       = var.service_name
  image              = "${module.ecr.repository_url}:latest"
  container_port     = var.container_port
  subnet_ids         = module.network.subnet_ids
  security_group_ids = [module.network.security_group_id]
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn
  log_group_name     = module.logging.log_group_name
  ecs_count          = var.ecs_count
  region             = var.aws_region

  db_host     = module.rds_mysql.db_endpoint
  db_port     = tostring(module.rds_mysql.db_port)
  db_name     = module.rds_mysql.db_name
  db_user     = module.rds_mysql.db_username
  db_password = module.rds_mysql.db_password

  dynamodb_table_name = module.dynamodb.table_name
  storage_backend     = "dynamodb"
}

