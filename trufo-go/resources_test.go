package trufo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newMockClient returns a client pointing at a test server running handler,
// authenticated with a gproduct-admin API key.
func newMockClient(t *testing.T, handler http.Handler) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClient(srv.URL, WithAPIKey("gproduct-admin:secret"))
	return c, srv.Close
}

func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decoding request body %s: %v", body, err)
	}
	return m
}

func TestInstanceCRUD(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(GPInstanceCreatePath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["gp_id"] != "gp_1" || b["name"] != "svc" {
			t.Errorf("create body = %v", b)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"gpi_id": "gpi_1"})
	})
	mux.HandleFunc(GPInstanceListPath, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"instances": []map[string]any{
			{"gpi_id": "gpi_1", "name": "svc", "record_id": "rec_1"},
		}})
	})
	mux.HandleFunc(GPInstanceInfoPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["gpi_id"] != "gpi_1" {
			t.Errorf("info body = %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"gpi_id": "gpi_1", "gp_id": "gp_1", "oid": "org_1", "name": "svc",
			"record_id": "rec_1", "created_at": "2026-11-20T11:26:48Z",
		})
	})
	mux.HandleFunc(GPInstanceEditPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["gpi_id"] != "gpi_1" || b["name"] != "svc2" {
			t.Errorf("edit body = %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc(GPInstanceDeletePath, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	c, cleanup := newMockClient(t, mux)
	defer cleanup()
	ctx := context.Background()

	id, err := c.CreateInstance(ctx, "gp_1", "svc")
	if err != nil || id != "gpi_1" {
		t.Fatalf("CreateInstance = %q, %v", id, err)
	}
	insts, err := c.ListInstances(ctx, "gp_1")
	if err != nil || len(insts) != 1 || insts[0].Name != "svc" {
		t.Fatalf("ListInstances = %+v, %v", insts, err)
	}
	inst, err := c.GetInstance(ctx, "gpi_1")
	if err != nil || inst.GPID != "gp_1" || inst.OID != "org_1" {
		t.Fatalf("GetInstance = %+v, %v", inst, err)
	}
	if err := c.RenameInstance(ctx, "gpi_1", "svc2"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteInstance(ctx, "gpi_1"); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialLifecycle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(GPCredentialRegisterPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["gpi_id"] != "gpi_1" || b["key_algorithm"] != "ES256" || b["public_key_pem"] != "PEM" {
			t.Errorf("register body = %v", b)
		}
		if _, ok := b["label"]; ok {
			t.Errorf("label should be omitted when empty: %v", b)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"gpic_id": "gpic_1"})
	})
	mux.HandleFunc(GPCredentialListPath, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"credentials": []map[string]any{
			{"gpic_id": "gpic_1", "gpi_id": "gpi_1", "key_algorithm": "ES256", "label": "l", "revoked": false},
		}})
	})
	mux.HandleFunc(GPCredentialEditPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["gpic_id"] != "gpic_1" || b["label"] != "new-label" {
			t.Errorf("edit body = %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc(GPCredentialRevokePath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["gpic_id"] != "gpic_1" {
			t.Errorf("revoke body = %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	c, cleanup := newMockClient(t, mux)
	defer cleanup()
	ctx := context.Background()

	id, err := c.RegisterCredential(ctx, "gpi_1", "PEM", KeyAlgorithmES256, "")
	if err != nil || id != "gpic_1" {
		t.Fatalf("RegisterCredential = %q, %v", id, err)
	}
	creds, err := c.ListCredentials(ctx, "gpi_1")
	if err != nil || len(creds) != 1 || creds[0].Revoked {
		t.Fatalf("ListCredentials = %+v, %v", creds, err)
	}
	if err := c.EditCredential(ctx, "gpic_1", "new-label"); err != nil {
		t.Fatal(err)
	}
	if err := c.RevokeCredential(ctx, "gpic_1"); err != nil {
		t.Fatal(err)
	}
}

func TestGProductAndCertAndAISystem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(GProductListPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if _, ok := b["oid"]; !ok {
			t.Errorf("oid is required by the schema: %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{"gproducts": []map[string]any{
			{"gp_id": "gp_1", "name": "prod", "pv_status": "approved", "pv_active": true},
		}})
	})
	mux.HandleFunc(CertListPath, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"certs": []map[string]any{
			{"serial_number": "AB12", "leaf_type": "c2pa-l1", "revocable": true, "trufo_managed": false},
		}})
	})
	mux.HandleFunc(CertRevokePath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["serial_number"] != "AB12" || b["revocation_reason"] != RevocationKeyCompromise {
			t.Errorf("revoke body = %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc(AISystemAddPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["name"] != "gen" || b["version"] != "1.0" {
			t.Errorf("add body = %v", b)
		}
		modes, _ := b["ai_involvement_modes"].([]any)
		if len(modes) != 1 || modes[0] != "ai_generated" {
			t.Errorf("modes = %v", modes)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"ai_system_id": "aisys_1"})
	})
	mux.HandleFunc(AISystemListPath, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{
			{"ai_system_id": "aisys_1", "name": "gen", "version": "1.0", "status": "active"},
		}})
	})
	mux.HandleFunc(AISystemStatusPath, func(w http.ResponseWriter, r *http.Request) {
		b := readBody(t, r)
		if b["ai_system_id"] != "aisys_1" || b["status"] != "inactive" {
			t.Errorf("status body = %v", b)
		}
		json.NewEncoder(w).Encode(map[string]any{"ai_system_id": "aisys_1", "status": "inactive"})
	})
	c, cleanup := newMockClient(t, mux)
	defer cleanup()
	ctx := context.Background()

	gps, err := c.ListGProducts(ctx, "org_1")
	if err != nil || len(gps) != 1 || !gps[0].PVActive {
		t.Fatalf("ListGProducts = %+v, %v", gps, err)
	}
	certs, err := c.ListCertificates(ctx)
	if err != nil || len(certs) != 1 || certs[0].SerialNumber != "AB12" {
		t.Fatalf("ListCertificates = %+v, %v", certs, err)
	}
	if err := c.RevokeCertificate(ctx, "AB12", RevocationKeyCompromise, ""); err != nil {
		t.Fatal(err)
	}
	id, err := c.AddAISystem(ctx, AISystem{Name: "gen", Version: "1.0", AIInvolvementModes: []string{AIInvolvementGenerated}})
	if err != nil || id != "aisys_1" {
		t.Fatalf("AddAISystem = %q, %v", id, err)
	}
	systems, err := c.ListAISystems(ctx)
	if err != nil || len(systems) != 1 {
		t.Fatalf("ListAISystems = %+v, %v", systems, err)
	}
	if _, err := c.SetAISystemStatus(ctx, "aisys_1", AISystemStatusInactive); err != nil {
		t.Fatal(err)
	}
}

func TestGetTaskUsesGET(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks/task_1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"task_id": "task_1", "status": "succeeded"})
	})
	c, cleanup := newMockClient(t, mux)
	defer cleanup()
	task, err := c.GetTask(context.Background(), "task_1")
	if err != nil || task.Status != "succeeded" {
		t.Fatalf("GetTask = %+v, %v", task, err)
	}
}
