# ── ECR repository for the processor image ───────────────────────────────────

resource "aws_ecr_repository" "processor" {
  name = "order-processor"
}

# ── CloudWatch log group ──────────────────────────────────────────────────────

resource "aws_cloudwatch_log_group" "processor" {
  name              = "/ecs/order-processor"
  retention_in_days = 7
}

# ── ECS Task Definition ───────────────────────────────────────────────────────

resource "aws_ecs_task_definition" "processor" {
  family                   = "order-processor-task"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn

  container_definitions = jsonencode([{
    name      = "order-processor-container"
    image     = "${aws_ecr_repository.processor.repository_url}:latest"
    essential = true

    environment = [
      { name = "SQS_QUEUE_URL",  value = aws_sqs_queue.orders.url },
      { name = "WORKER_COUNT",   value = tostring(var.worker_count) }
    ]

    portMappings = [{
      containerPort = 8080
    }]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.processor.name
        "awslogs-region"        = var.aws_region
        "awslogs-stream-prefix" = "ecs"
      }
    }
  }])
}

# ── ECS Service (worker — no ALB, no public IP) ───────────────────────────────

resource "aws_ecs_service" "processor" {
  name            = "order-processor"
  cluster         = module.ecs.cluster_id
  task_definition = aws_ecs_task_definition.processor.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = module.vpc.private_subnet_ids
    security_groups  = [aws_security_group.ecs_sg.id]
    assign_public_ip = false
  }
}
