data "aws_vpc" "default" {
  default = true
}

resource "random_password" "db_password" {
  length  = 20
  special = false
}

resource "aws_db_subnet_group" "this" {
  name       = "hw8-rds-subnets"
  subnet_ids = var.subnet_ids
}

resource "aws_security_group" "this" {
  name        = "hw8-rds-sg"
  description = "MySQL access from ECS tasks only"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description     = "MySQL from ECS tasks"
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [var.ecs_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }
}

resource "aws_db_instance" "this" {
  identifier = "hw8-mysql"

  engine         = "mysql"
  engine_version = var.engine_version

  instance_class    = var.db_instance_class
  allocated_storage = var.allocated_storage
  storage_type      = "gp2"

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.this.id]

  username = var.db_username
  password = random_password.db_password.result
  db_name  = var.db_name

  port = 3306

  publicly_accessible     = true
  multi_az                = false
  backup_retention_period = var.backup_retention_period

  skip_final_snapshot = var.skip_final_snapshot
  deletion_protection = var.deletion_protection

  apply_immediately = var.apply_immediately
}

