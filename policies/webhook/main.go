package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
)

type AdmissionRequest struct {
	Request struct {
		UID       string                 `json:"uid"`
		Kind      struct{ Kind string }  `json:"kind"`
		Namespace string                 `json:"namespace"`
		Object    map[string]interface{} `json:"object"`
	} `json:"request"`
}

type AdmissionResponse struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Response   struct {
		UID     string `json:"uid"`
		Allowed bool   `json:"allowed"`
		Status  *struct {
			Message string `json:"message"`
		} `json:"status,omitempty"`
	} `json:"response"`
}

// getPackageName reads the package declaration from a .rego file
func getPackageName(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return strings.TrimPrefix(line, "package ")
		}
	}
	return ""
}

func loadPolicies(dir string) []string {
	files, _ := filepath.Glob(filepath.Join(dir, "*.rego"))
	return files
}

func evaluate(policyFiles []string, input map[string]interface{}) ([]string, error) {
	var denials []string
	ctx := context.Background()

	for _, pf := range policyFiles {
		data, err := os.ReadFile(pf)
		if err != nil {
			continue
		}

		pkg := getPackageName(pf)
		if pkg == "" {
			continue
		}

		// Convert package path to query: k8s.no_latest_tag -> data.k8s.no_latest_tag.deny
		query := fmt.Sprintf("data.%s.deny", pkg)

		r := rego.New(
			rego.Query(query),
			rego.Module(pf, string(data)),
			rego.Input(input),
			rego.SetRegoVersion(ast.RegoV1),
		)

		rs, err := r.Eval(ctx)
		if err != nil {
			log.Printf("eval error for %s: %v", pf, err)
			continue
		}

		if len(rs) == 0 || len(rs[0].Expressions) == 0 {
			continue
		}

		switch v := rs[0].Expressions[0].Value.(type) {
		case []interface{}:
			for _, msg := range v {
				if s, ok := msg.(string); ok {
					denials = append(denials, s)
				}
			}
		case map[string]interface{}:
			for msg := range v {
				denials = append(denials, msg)
			}
		}
	}
	return denials, nil
}

func webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var admReq AdmissionRequest
	if err := json.Unmarshal(body, &admReq); err != nil {
		http.Error(w, "failed to parse request", http.StatusBadRequest)
		return
	}

	input := map[string]interface{}{
		"request": map[string]interface{}{
			"uid":       admReq.Request.UID,
			"kind":      map[string]interface{}{"kind": admReq.Request.Kind.Kind},
			"namespace": admReq.Request.Namespace,
			"object":    admReq.Request.Object,
		},
	}

	policyDir := os.Getenv("POLICY_DIR")
	if policyDir == "" {
		policyDir = "../../rego"
	}

	denials, _ := evaluate(loadPolicies(policyDir), input)

	resp := AdmissionResponse{
		APIVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
	}
	resp.Response.UID = admReq.Request.UID

	if len(denials) == 0 {
		resp.Response.Allowed = true
		log.Printf("ALLOW %s/%s", admReq.Request.Namespace, admReq.Request.Kind.Kind)
	} else {
		resp.Response.Allowed = false
		msg := "FinOps policy violations:\n"
		for _, d := range denials {
			msg += fmt.Sprintf("  - %s\n", d)
		}
		resp.Response.Status = &struct {
			Message string `json:"message"`
		}{Message: msg}
		log.Printf("DENY %s/%s: %v", admReq.Request.Namespace, admReq.Request.Kind.Kind, denials)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8443"
	}

	http.HandleFunc("/validate", webhook)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("OPA webhook listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
