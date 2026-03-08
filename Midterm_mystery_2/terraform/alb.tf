
locals {
  vpc_id            = "vpc-05b6f59630c3b4e68"
  public_subnet_ids = [
    "subnet-0b3f7aeaf7d8c288a", # us-east-1c
    "subnet-0ce0c0504b2298593", # us-east-1d
    "subnet-0307d2b0a1fdb993b", # us-east-1b
    "subnet-02c40913c9aa94783", # us-east-1f
    "subnet-056e342152fb8236b", # us-east-1e
    "subnet-01108d86569b4f5e6", # us-east-1a
  ]
  ecs_cluster_name = "CS6650L2-cluster"
  ecs_service_name = "CS6650L2"
  container_name   = "CS6650L2-container"
}

# Security Group: ALB
resource "aws_security_group" "alb_sg" {
  name        = "CS6650L2-alb-sg"
  description = "Allow HTTP traffic to ALB"
  vpc_id      = local.vpc_id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "CS6650L2-alb-sg" }
}

# Security Group: ECS Tasks (allow traffic from ALB) 
resource "aws_security_group" "ecs_sg" {
  name        = "CS6650L2-ecs-sg"
  description = "Allow traffic from ALB to ECS tasks"
  vpc_id      = local.vpc_id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "CS6650L2-ecs-sg" }
}

# Application Load Balancer 
resource "aws_lb" "app_alb" {
  name               = "CS6650L2-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = local.public_subnet_ids

  tags = { Name = "CS6650L2-alb" }
}

# Target Group
resource "aws_lb_target_group" "app_tg" {
  name        = "CS6650L2-tg"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = local.vpc_id
  target_type = "ip" # Required for Fargate

  health_check {
    path                = "/health"
    interval            = 30
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    matcher             = "200"
  }

  tags = { Name = "CS6650L2-tg" }
}

# ALB Listener
resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.app_alb.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app_tg.arn
  }
}

# Auto Scaling Target
resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = 4
  min_capacity       = 2
  resource_id        = "service/${local.ecs_cluster_name}/${local.ecs_service_name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
  depends_on = [module.ecs]

}

# Auto Scaling Policy (Target Tracking: 70% CPU) 
resource "aws_appautoscaling_policy" "cpu_policy" {
  name               = "CS6650L2-cpu-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_target.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value       = 70.0 # Scale out when avg CPU > 70%
    scale_in_cooldown  = 300  # Wait 5 min before scaling in
    scale_out_cooldown = 300  # Wait 5 min before scaling out
  }
}

# Output
output "alb_dns_name" {
  description = "Use this as your Locust host for Part III load testing"
  value       = "http://${aws_lb.app_alb.dns_name}"
}