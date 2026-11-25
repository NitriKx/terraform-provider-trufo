variable "generator_product_id" {
  description = "ID (gp_...) of the generator product (created in the Trufo dashboard) to look up with the trufo-ai_generator_product data source. Set this OR generator_product_name, not both. The product needs an active Product Validation before instances can be created."
  type        = string
  default     = ""
}

variable "generator_product_name" {
  description = "Name of the generator product to look up with the trufo-ai_generator_product data source (used when generator_product_id is empty). The product needs an active Product Validation before instances can be created."
  type        = string
}

variable "instance_name" {
  description = "Human-readable name of the C2PA signing instance (trufo-ai_instance) created under the generator product."
  type        = string
  default     = "c2pa-signing-1"
}

variable "credential_public_key_path" {
  description = <<-EOT
    Path to the PEM-encoded PUBLIC key to enroll on the instance (ES256 / EC P-256).
    Generate the key pair locally and keep the private key out of Terraform:

      openssl ecparam -name prime256v1 -genkey -noout -out instance-key.pem
      openssl ec -in instance-key.pem -pubout -out instance-public-key.pem

    The private key (instance-key.pem) is never passed to Terraform — it only
    signs the CSR-JWT client assertion used for certificate enrollment later.
  EOT
  type        = string
  default     = "instance-public-key.pem"
}

variable "credential_label" {
  description = "Human-readable label for the enrolled credential (trufo-ai_instance_credential). Use a new label per rotation, e.g. primary-2026-11, primary-2027-05."
  type        = string
  default     = "primary-2026-11"
}
