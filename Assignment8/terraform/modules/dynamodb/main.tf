resource "aws_dynamodb_table" "shopping_carts" {
  name         = var.table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "cart_id"

  attribute {
    name = "cart_id"
    type = "S"
  }

  tags = {
    Name = var.table_name
  }
}
