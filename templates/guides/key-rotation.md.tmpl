---
page_title: "Key Rotation - trufo"
subcategory: "Guides"
description: |-
  How to rotate an instance credential (enrolled public key) safely.
---

# Key Rotation

An instance may hold at most **two active credentials** — that cap is what makes rotation possible.
The rotation flow is: register the replacement, cut over, then revoke the old one.

```terraform
# 1. Add the new credential alongside the old one (two active credentials is the maximum).
resource "trufo_instance_credential" "next" {
  gpi_id         = trufo_instance.signing.id
  public_key_pem = file("instance-public-key-2027.pem")
  key_algorithm  = "ES256"
  label          = "primary-2027-05"
}

# 2. Keep the old one until your signer has cut over:
resource "trufo_instance_credential" "current" {
  gpi_id         = trufo_instance.signing.id
  public_key_pem = file("instance-public-key.pem")
  key_algorithm  = "ES256"
  label          = "primary-2026-11"
}
```

Apply, point your signing service at the new private key (the one matching
`instance-public-key-2027.pem` — private keys never enter Terraform), verify enrollment works
(a certificate request via `POST /ra/c2pa/csr-jwt` signed with the new key), then remove the old
resource from the configuration and apply again. Removing the resource **revokes** the old credential
permanently.

If you hit the two-credential cap, revoke one first (`terraform destroy -target=...` or the dashboard).
