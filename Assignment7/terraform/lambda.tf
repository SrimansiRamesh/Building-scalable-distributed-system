# ── Lambda Function ───────────────────────────────────────────────────────────

resource "aws_lambda_function" "order_processor" {
  filename         = "../lambda/function.zip"
  function_name    = "order-processor-lambda"
  role             = data.aws_iam_role.lab_role.arn
  handler          = "bootstrap"        # required name for provided.al2
  runtime          = "provided.al2"
  memory_size      = 512
  timeout          = 30                 # must exceed 3s processing time

  # Forces Terraform to redeploy when the zip changes
  source_code_hash = filebase64sha256("../lambda/function.zip")
}

# ── Allow SNS to invoke the Lambda ───────────────────────────────────────────

resource "aws_lambda_permission" "sns_invoke" {
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.order_processor.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.orders.arn
}

# ── Subscribe Lambda directly to the SNS topic ────────────────────────────────
# SNS now fans out to both SQS (ECS workers) and Lambda simultaneously.
# In a pure serverless setup you would remove the SQS subscription.

resource "aws_sns_topic_subscription" "lambda" {
  topic_arn = aws_sns_topic.orders.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.order_processor.arn

  depends_on = [aws_lambda_permission.sns_invoke]
}

# ── Output ────────────────────────────────────────────────────────────────────

output "lambda_function_name" {
  value = aws_lambda_function.order_processor.function_name
}

output "lambda_log_group" {
  description = "CloudWatch log group for Lambda — look here for cold start REPORT lines"
  value       = "/aws/lambda/${aws_lambda_function.order_processor.function_name}"
}
