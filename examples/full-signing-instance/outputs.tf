output "generator_product_id" {
  description = "Generator product ID (gp_…) the instance belongs to."
  value       = data.trufo-ai_generator_product.c2pa.id
}

output "instance_id" {
  description = "Instance ID (gpi_…)."
  value       = trufo-ai_instance.signing.id
}

output "instance_name" {
  description = "Name of the signing instance."
  value       = trufo-ai_instance.signing.name
}

output "credential_id" {
  description = "Enrolled credential ID (gpic_…). Import an existing credential with: terraform import trufo-ai_instance_credential.primary {gpi_id}/{gpic_id}"
  value       = trufo-ai_instance_credential.primary.id
}

output "credential_label" {
  description = "Label of the enrolled credential."
  value       = trufo-ai_instance_credential.primary.label
}

output "credential_revoked" {
  description = "Whether the credential has been revoked (removing the resource revokes it permanently)."
  value       = trufo-ai_instance_credential.primary.revoked
}
