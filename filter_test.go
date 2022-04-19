package main

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/apigateway"
)

func Test_FilterServices(t *testing.T) {
	filters := []string{
		"Name=tag:STAGE,Values=dev",
		"Name=tag:env,Values=dev",
	}

	exclusionFilters := []string{
		"Name=tag:service,Values=bad-service-dev-env",
	}

	devStageService := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: stringP("dev-stage-api"),
			Tags: map[string]*string{
				"STAGE": stringP("dev"),
			},
		},
	}

	devEnvService := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: stringP("dev-env-api"),
			Tags: map[string]*string{
				"env": stringP("dev"),
			},
		},
	}

	prdEnvService := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: stringP("prd-env-api"),
			Tags: map[string]*string{
				"env": stringP("prd"),
			},
		},
	}

	blankService := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: stringP("blank-api"),
			Tags: map[string]*string{},
		},
	}

	devEnvServiceToBeFiltered := &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: stringP("bad-service-dev-env"),
			Tags: map[string]*string{
				"env":     stringP("dev"),
				"service": stringP("bad-service-dev-env"),
			},
		},
	}

	services := []*APIGatewayService{
		devStageService,
		devEnvService,
		devEnvServiceToBeFiltered,
		prdEnvService,
		blankService,
	}

	result := FilterServices(services, filters, exclusionFilters)
	expectedResult := []*APIGatewayService{
		devStageService,
		devEnvService,
	}

	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Expected %v, got %v", expectedResult, result)
	}
}
