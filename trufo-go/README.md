# trufo-go

Go client for the [Trufo](https://trufo.ai) API (`api.trufo.ai`), ported from the official
[Python SDK trufo-py](https://github.com/trufo-ai/trufo-py).

Scope: account/device/loopback auth flows and the Registration Authority resources — generator products,
instances, instance credentials, certificates, AI systems, tasks. Media-processing endpoints (C2PA signing,
watermarking, EST enrollment) are intentionally out of scope.

## Usage

```go
// API-key mode (X-API-Key: {scope}:{key})
client := trufo.NewClient(trufo.APIURLTest, trufo.WithAPIKey(os.Getenv("TRUFO_API_KEY")))

// Bearer mode with automatic refresh on 401
client := trufo.NewClient(trufo.APIURL, trufo.WithTokens(access, refresh))

// Interactive login with a trufo-api scoped key (loopback browser flow, or
// device flow on headless machines)
tokens, err := client.DeviceFlow(ctx, func(uri, code string) {
    fmt.Printf("Visit %s and confirm %s\n", uri, code)
})

gpiID, err := client.CreateInstance(ctx, "gp_...", "signing-service-eu-1")
gpicID, err := client.RegisterCredential(ctx, gpiID, publicKeyPEM, trufo.KeyAlgorithmES256, "primary")
```

All endpoints are `POST` with a JSON body (except `GET /tasks/{id}`); errors are returned as `*trufo.APIError`
carrying the HTTP status, the stable `detail` code, any 422 field errors, and the `X-Request-ID` header.

## License

Apache-2.0 — see [LICENSE](../LICENSE).
