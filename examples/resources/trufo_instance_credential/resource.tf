resource "trufo_instance" "signing_service" {
  gp_id = "gp_..."
  name  = "signing-service-eu-1"
}

# Enroll a public key on the instance. The private key stays on your side and
# signs the client assertion used for certificate enrollment. An instance
# holds at most two active credentials — rotate by adding the replacement
# first, then removing the old one (delete = revoke).
resource "trufo_instance_credential" "primary" {
  gpi_id         = trufo_instance.signing_service.id
  public_key_pem = file("instance-public-key.pem")
  key_algorithm  = "ES256" # EC P-256; EdDSA (Ed25519) also accepted
  label          = "primary-2026"
}
