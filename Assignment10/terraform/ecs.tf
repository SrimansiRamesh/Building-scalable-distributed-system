variable "image_url" {
  description = "ECR image URL for the KV node"
  type        = string
}

variable "w" {
  description = "Write quorum (1, 3, or 5)"
  type        = number
  default     = 5
}

variable "r" {
  description = "Read quorum (1, 3, or 5)"
  type        = number
  default     = 1
}

variable "mode" {
  description = "leader-follower or leaderless"
  type        = string
  default     = "leader-follower"
}

# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = "a10-cluster"
}

# CloudWatch log group
resource "aws_cloudwatch_log_group" "nodes" {
  name              = "/ecs/a10-nodes"
  retention_in_days = 1
}

# ── Network Load Balancers for each node ─────────────────────────────────────
# Since we can't use service discovery, each node gets its own NLB
# so other nodes can reach it via a stable DNS name.
# node-0 = leader (also behind the main ALB for client traffic)
# node-1 through node-4 = followers

resource "aws_lb" "nodes" {
  for_each           = toset(["node-0", "node-1", "node-2", "node-3", "node-4"])
  name               = "a10-${each.key}-nlb"
  internal           = true
  load_balancer_type = "network"
  subnets            = [aws_subnet.private_a.id, aws_subnet.private_b.id]
  tags               = { Name = "a10-${each.key}-nlb" }
}

resource "aws_lb_target_group" "nodes" {
  for_each    = toset(["node-0", "node-1", "node-2", "node-3", "node-4"])
  name        = "a10-${each.key}-tg"
  port        = 8080
  protocol    = "TCP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    path                = "/health"
    protocol            = "HTTP"
    interval            = 15
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  tags = { Name = "a10-${each.key}-tg" }
}

resource "aws_lb_listener" "nodes" {
  for_each          = toset(["node-0", "node-1", "node-2", "node-3", "node-4"])
  load_balancer_arn = aws_lb.nodes[each.key].arn
  port              = 8080
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.nodes[each.key].arn
  }
}

# ── Peer address locals ───────────────────────────────────────────────────────
locals {
  node_ids     = ["node-0", "node-1", "node-2", "node-3", "node-4"]
  follower_ids = ["node-1", "node-2", "node-3", "node-4"]

  # Each node's internal NLB DNS — used as peer addresses
  node_dns = {
    for id in local.node_ids :
    id => "http://${aws_lb.nodes[id].dns_name}:8080"
  }

  # Leader peers = all follower NLB addresses
  leader_peers = join(",", [for id in local.follower_ids : local.node_dns[id]])

  # Leaderless: each node's peers = all OTHER nodes
  leaderless_peers = {
    for id in local.node_ids :
    id => join(",", [for other in local.node_ids : local.node_dns[other] if other != id])
  }
}

# ── ECS Task Definitions ──────────────────────────────────────────────────────
resource "aws_ecs_task_definition" "node" {
  for_each = toset(local.node_ids)

  family                   = "a10-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = data.aws_iam_role.lab_role.arn
  task_role_arn            = data.aws_iam_role.lab_role.arn

  container_definitions = jsonencode([{
    name  = "kv-node"
    image = var.image_url

    portMappings = [{
      containerPort = 8080
      protocol      = "tcp"
    }]

    environment = [
      { name = "NODE_ID", value = each.key },
      {
        name  = "ROLE"
        value = var.mode == "leaderless" ? "leaderless" : (each.key == "node-0" ? "leader" : "follower")
      },
      { name = "W", value = tostring(var.w) },
      { name = "R", value = tostring(var.r) },
      {
        name  = "PEERS"
        value = var.mode == "leaderless" ? local.leaderless_peers[each.key] : (each.key == "node-0" ? local.leader_peers : "")
      },
      { name = "PORT", value = "8080" }
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.nodes.name
        "awslogs-region"        = "us-east-1"
        "awslogs-stream-prefix" = each.key
      }
    }

    healthCheck = {
      command     = ["CMD-SHELL", "wget -qO- http://localhost:8080/health || exit 1"]
      interval    = 15
      timeout     = 5
      retries     = 3
      startPeriod = 10
    }
  }])

  depends_on = [aws_lb.nodes]
}

# ── ECS Services ──────────────────────────────────────────────────────────────
resource "aws_ecs_service" "node" {
  for_each = toset(local.node_ids)

  name            = "a10-${each.key}"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.node[each.key].arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_groups  = [aws_security_group.nodes.id]
    assign_public_ip = false
  }

  # Attach each node to its own internal NLB target group
  load_balancer {
    target_group_arn = aws_lb_target_group.nodes[each.key].arn
    container_name   = "kv-node"
    container_port   = 8080
  }

  # node-0 also gets attached to the public ALB for client traffic
  dynamic "load_balancer" {
    for_each = each.key == "node-0" ? [1] : []
    content {
      target_group_arn = aws_lb_target_group.leader.arn
      container_name   = "kv-node"
      container_port   = 8080
    }
  }

  depends_on = [aws_lb_listener.nodes, aws_lb_listener.main]
}

# ── Outputs ───────────────────────────────────────────────────────────────────
output "leader_alb_dns" {
  value = "http://${aws_lb.main.dns_name}"
}

output "node_internal_dns" {
  value = local.node_dns
}