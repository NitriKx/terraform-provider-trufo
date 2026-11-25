//go:build tools

// Package tools pins tool dependencies (tfplugindocs) so they are tracked in
// go.mod.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
