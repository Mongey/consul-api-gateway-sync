package main

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/apigateway"
)

func Test_TagsReplacement(t *testing.T) {
	stage := "staging"
	logicalID := "ApiGatewayRestApi"
	stackID := "arn:aws:cloudformation:us-west-2:1234567:stack/my-service-" + stage + "/7fd50290-84eb-11ec-93c7-0ac7bf603f03"
	serviceName := stage + "-my-service"
	service := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: &serviceName,
			Tags: map[string]*string{
				"STAGE":                         &stage,
				"aws:cloudformation:logical-id": &logicalID,
				"aws:cloudformation:stack-id":   &stackID,
			},
		},
	}

	expectedResult := map[string]string{
		"STAGE":                         stage,
		"aws-cloudformation-logical-id": logicalID,
		"aws-cloudformation-stack-id":   "arn-aws-cloudformation-us-west-2-1234567-stack-my-service-staging-7fd50290-84eb-11ec-93c7-0ac7bf603f03",
	}
	result := service.Tags()

	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Got '%s', expected '%s'", result, expectedResult)
	}
}

func Test_NameParsing(t *testing.T) {
	stage := "staging"
	serviceName := "staging-my-service"
	service := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: &serviceName,
			Tags: map[string]*string{
				"STAGE": &stage,
			},
		},
	}

	expectedName := "my-service"
	result := service.Name()

	if result != expectedName {
		t.Errorf("Got '%s', expected '%s'", result, expectedName)
	}
}

func Test_TagsTemplate(t *testing.T) {
	tagTemplates := []string{
		"traefik-dev-private.enabled=true",
		"traefik-dev-private.tags=private",
		"traefik-dev-private.entryPoints=http",
		"traefik-dev-private.frontend.rule=Host: {{ .Name }}.example.org; AddPrefix: /{{index .Tags \"STAGE\" }}/",
	}

	expectedResult := []string{
		"traefik-dev-private.enabled=true",
		"traefik-dev-private.tags=private",
		"traefik-dev-private.entryPoints=http",
		"traefik-dev-private.frontend.rule=Host: my-service.example.org; AddPrefix: /staging/",
	}

	stage := "staging"
	serviceName := "staging-my-service"
	service := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: &serviceName,
			Tags: map[string]*string{
				"STAGE": &stage,
			},
		},
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
