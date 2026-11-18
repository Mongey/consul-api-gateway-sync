package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	consulapi "github.com/hashicorp/consul/api"
	hclog "github.com/hashicorp/go-hclog"
)

func TestRegisterServices(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/restapis":
			_, _ = w.Write([]byte(`{"items":[{"id":"api-123","name":"staging-orders","tags":{"STAGE":"staging","env":"test"}}]}`))
		case "/restapis/api-123/stages":
			_, _ = w.Write([]byte(`{"item":[{"stageName":"staging"}]}`))
		default:
			t.Errorf("unexpected API Gateway request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer gateway.Close()
	registrations := make(chan consulapi.CatalogRegistration, 1)
	consul := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/catalog/register" {
			t.Errorf("unexpected Consul request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var registration consulapi.CatalogRegistration
		if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
			t.Error(err)
			http.Error(w, "invalid registration", http.StatusBadRequest)
			return
		}
		registrations <- registration
		_, _ = w.Write([]byte("true"))
	}))
	defer consul.Close()
	consulClient, err := consulapi.NewClient(&consulapi.Config{Address: consul.URL})
	if err != nil {
		t.Fatal(err)
	}
	c := &Client{
		flags:  []string{"Name=tag:env,Values=^test$"},
		tags:   []string{"stage={{index .StageNames 0}}", "env={{index .Tags \"env\"}}"},
		region: "us-west-2",
		logger: hclog.NewNullLogger(),
		svc: apigateway.NewFromConfig(aws.Config{
			Region:      "us-west-2",
			Credentials: credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
		}, func(o *apigateway.Options) { o.BaseEndpoint = aws.String(gateway.URL) }),
		consulClient: consulClient,
	}
	if err := c.registerServices(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-registrations:
		if got.Node != "api-123" || got.Address != "api-123.execute-api.us-west-2.amazonaws.com" || got.Service.Service != "orders" {
			t.Fatalf("unexpected registration: %+v", got)
		}
		if len(got.Service.Tags) != 2 || got.Service.Tags[0] != "stage=staging" || got.Service.Tags[1] != "env=test" {
			t.Fatalf("unexpected service tags: %v", got.Service.Tags)
		}
	default:
		t.Fatal("no service registered with Consul")
	}
}

func TestNewClientReturnsConfigError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("[profile regression]\nrequest_min_compression_size_bytes = invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", path)
	t.Setenv("AWS_PROFILE", "regression")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	client, err := NewClient(nil, nil, nil, "us-west-2", hclog.NewNullLogger())
	if err == nil || client != nil {
		t.Fatalf("expected AWS config error, got client %v, error %v", client, err)
	}
}
