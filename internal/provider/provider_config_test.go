package provider

import (
	"testing"

	trufo "github.com/NitriKx/terraform-trufo-ai-provider/trufo-go"
)

func TestResolveAPIURL(t *testing.T) {
	cases := []struct {
		apiURL, region string
		want           string
		wantErr        bool
	}{
		{"", "", trufo.APIURL, false},                                   // default: production global
		{"", "global", trufo.APIURL, false},                             // explicit prod
		{"", "eu", trufo.APIURLEurope, false},                           // EU prod
		{"", "test", trufo.APIURLTest, false},                           // test API
		{"https://custom.example", "", "https://custom.example", false}, // api_url overrides region
		{"https://custom.example", "test", "https://custom.example", false},
		{"", "mars", "", true}, // invalid region
	}
	for _, c := range cases {
		got, err := resolveAPIURL(c.apiURL, c.region)
		if c.wantErr {
			if err == nil {
				t.Errorf("resolveAPIURL(%q, %q): expected error, got %q", c.apiURL, c.region, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("resolveAPIURL(%q, %q) = %q, %v; want %q", c.apiURL, c.region, got, err, c.want)
		}
	}
}
