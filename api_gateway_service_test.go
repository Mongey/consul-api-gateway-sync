package main

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/apigateway"
)

func stringP(v string) *string { return &v }

func Test_TagsReplacement(t *testing.T) {
	stage := "staging"
	logicalID := "ApiGatewayRestApi"
	serviceName := stage + "-my-service"
	service := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: &serviceName,
			Tags: map[string]*string{
				"STAGE":                         stringP(stage),
				"aws:cloudformation:logical-id": stringP(logicalID),
				"aws:cloudformation:stack-id":   stringP("arn:aws:cloudformation:us-west-2:1234567:stack/my-service-" + stage + "/7fd50290-84eb-11ec-93c7-0ac7bf603f03"),
			},
		},
	}

	expectedResult := map[string]string{
		"STAGE":                         stage,
		"aws:cloudformation:logical-id": logicalID,
		"aws:cloudformation:stack-id":   "arn:aws:cloudformation:us-west-2:1234567:stack/my-service-staging/7fd50290-84eb-11ec-93c7-0ac7bf603f03",
	}

	result := service.Tags()

	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Got '%s', expected '%s'", result, expectedResult)
	}
}

func Test_NameParsing(t *testing.T) {
	tests := map[string]struct {
		expectedServiceName string
		service             *APIGatewayService
	}{
		"stage prefix replacement": {
			"my-service",
			&APIGatewayService{
				region: "us-west-2",
				restAPI: &apigateway.RestApi{
					Name: stringP("staging-my-service"),
					Tags: map[string]*string{
						"STAGE": stringP("staging"),
					},
				},
			},
		},
		"stage suffix replacement": {
			"my-service",
			&APIGatewayService{
				region: "us-west-2",
				restAPI: &apigateway.RestApi{
					Name: stringP("my-service-staging"),
					Tags: map[string]*string{
						"STAGE": stringP("staging"),
					},
				},
			},
		},
		"service name use": {
			"my-service",
			&APIGatewayService{
				region: "us-west-2",
				restAPI: &apigateway.RestApi{
					Name: stringP("staging-my-service"),
					Tags: map[string]*string{
						"service": stringP("my-service"),
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tc.service.Name()
			if result != tc.expectedServiceName {
				t.Errorf("Got '%s', expected '%s'", result, tc.expectedServiceName)
			}
		})
	}
}

func Test_TagsTemplate(t *testing.T) {
	tagTemplates := []string{
		"traefik-dev-private.enabled=true",
		"traefik-dev-private.tags=private",
		"traefik-dev-private.entryPoints=http",
		"traefik-dev-private.frontend.rule=Host: {{ .Name }}.example.org; AddPrefix: /{{index .Tags \"STAGE\" }}/",
		"traefik-dev-public.frontend.rule=Host: {{ .Name }}.example.org; AddPrefix: /{{index .StageNames 0 }}/",
		"{{index .Tags \"A key that does not exist\" }}",
	}

	expectedResult := []string{
		"traefik-dev-private.enabled=true",
		"traefik-dev-private.tags=private",
		"traefik-dev-private.entryPoints=http",
		"traefik-dev-private.frontend.rule=Host: my-service.example.org; AddPrefix: /staging/",
		"traefik-dev-public.frontend.rule=Host: my-service.example.org; AddPrefix: /stg/",
	}

	service := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: stringP("staging-my-service"),
			Tags: map[string]*string{
				"STAGE": stringP("staging"),
			},
		},
		StageNames: []string{"stg", "uat"},
	}

	result := service.TagsFromTemplate(tagTemplates)

	if len(result) != len(expectedResult) {
		t.Errorf("Expected %d, got %d", len(expectedResult), len(result))

	}
	for i, r := range result {
		if r != expectedResult[i] {
			t.Errorf("Expected '%s', got '%s'", expectedResult[i], r)
		}
	}
}
