# AI systems are registered on the production hosts with a c2pa-sign-* or
# watermark-* scoped API key (or a Bearer token with the c2pa_sign permission).
resource "trufo_ai_system" "image_gen" {
  name                        = "acme-image-gen"
  version                     = "2.1.0"
  ai_involvement_modes        = ["ai_generated"]
  c2pa_first_party_attestable = true # requires the Created Assertion Acknowledgement
  nickname                    = "Image Gen v2"
  status                      = "active"

  c2pa_ai_disclosure = {
    modelType      = "c2pa.types.model"
    modelName      = "acme-image-gen-2.1"
    contentProfile = { humanOversightLevel = "prompt_guided" }
  }
}
