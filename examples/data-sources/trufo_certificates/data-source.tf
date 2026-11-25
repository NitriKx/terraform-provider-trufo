data "trufo_certificates" "all" {}

output "active_certs" {
  value = [for c in data.trufo_certificates.all.certs : c.serial_number if c.revocation_status == "good"]
}
