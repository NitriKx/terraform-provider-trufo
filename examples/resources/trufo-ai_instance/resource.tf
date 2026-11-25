data "trufo-ai_generator_product" "by_name" {
  name = "my-generator-product"
}

resource "trufo-ai_instance" "signing_service" {
  gp_id = data.trufo-ai_generator_product.by_name.id
  name  = "signing-service-eu-1"
}
