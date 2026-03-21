output "alb_dns_name" {
  description = "Use this as your Locust host"
  value       = "http://${aws_lb.app_alb.dns_name}"
}

output "ecr_repository_url" {
  description = "Push your order-api image here"
  value       = module.ecr.repository_url
}

output "processor_ecr_repository_url" {
  description = "Push your order-processor image here"
  value       = aws_ecr_repository.processor.repository_url
}

output "sns_topic_arn" {
  value = aws_sns_topic.orders.arn
}

output "sqs_queue_url" {
  value = aws_sqs_queue.orders.url
}

output "ecs_cluster_name" {
  value = module.ecs.cluster_name
}

output "ecs_service_name" {
  value = module.ecs.service_name
}
