data "trufo_generator_product" "by_name" {
  name = "my-generator-product"
}

resource "trufo_instance" "signing_service" {
  gp_id = data.trufo_generator_product.by_name.id
  name  = "signing-service-eu-1"
}
