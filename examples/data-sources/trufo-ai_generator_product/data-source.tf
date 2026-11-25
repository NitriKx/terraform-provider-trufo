data "trufo-ai_generator_product" "by_id" {
  gp_id = "gp_..."
}

data "trufo-ai_generator_product" "by_name" {
  name = "my-generator-product"
  oid  = "org_..." # required by /gproduct/list; the server scopes to your credential's org
}

output "pv_active" {
  value = data.trufo-ai_generator_product.by_id.pv_active
}
