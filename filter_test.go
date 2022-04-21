package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/apigateway"
)

func ptrMap(m map[string]string) map[string]*string {
	result := make(map[string]*string)
	for k, v := range m {
		copiedValue := strings.Clone(v)
		result[k] = &copiedValue
	}
	return result
}

func nonptrMap(m map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = *v
	}
	return result
}

func merge(a, b map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}

	return result
}

func newTestService(name string, tags map[string]string) *APIGatewayService {
	defaultTags := map[string]string{"service": name}
	newTags := merge(defaultTags, tags)

	return &APIGatewayService{
		region: "us-west-2",
		restAPI: &apigateway.RestApi{
			Name: &name,
			Tags: ptrMap(newTags),
		},
	}
}

type testData struct {
	filters          []string
	exclusionFilters []string
	services         []*APIGatewayService
	expectedResult   []*APIGatewayService
}

func Test_exclusions(t *testing.T) {
	devEnvService := newTestService(
		"dev-env-api",
		map[string]string{
			"env":     "dev",
			"service": "dev-env-api",
		},
	)

	serviceToBeExcluded := newTestService(
		"service-to-be-excluded",
		map[string]string{
			"env":     "dev",
			"service": "service-to-be-excluded",
		},
	)

	production := newTestService(
		"my-prouction-api",
		map[string]string{
			"env": "production",
		},
	)

	tcs := []struct {
		description string
		testData    testData
	}{
		{
			"simple exclusion case",
			testData{
				filters: []string{},
				exclusionFilters: []string{
					fmt.Sprintf("Name=tag:service,Values=%s", "service-to-be-excluded"),
				},
				services: []*APIGatewayService{
					devEnvService,
					serviceToBeExcluded,
				},
				expectedResult: []*APIGatewayService{
					devEnvService,
				},
			},
		},
		{
			"multiple exclusion filters",
			testData{
				filters: []string{},
				exclusionFilters: []string{
					fmt.Sprintf("Name=tag:env,Values=%s", "dev"),
					fmt.Sprintf("Name=tag:env,Values=%s", "prod"),
				},
				services: []*APIGatewayService{
					newTestService(
						"service-to-be-excluded",
						map[string]string{
							"env": "dev",
						},
					),
					newTestService(
						"service-to-be-excluded",
						map[string]string{
							"env": "prod",
						},
					),
				},
				expectedResult: []*APIGatewayService{},
			},
		},
		{
			"simple filter",
			testData{
				filters: []string{
					fmt.Sprintf("Name=tag:env,Values=%s", "dev"),
				},
				exclusionFilters: []string{},
				services: []*APIGatewayService{
					devEnvService,
					newTestService(
						"service-to-be-excluded",
						map[string]string{
							"env": "production",
						},
					),
					newTestService(
						"service-to-be-excluded",
						map[string]string{
							"env": "staging",
						},
					),
				},
				expectedResult: []*APIGatewayService{
					devEnvService,
				},
			},
		},
		{
			"A regex filter",
			testData{
				filters: []string{
					fmt.Sprintf("Name=tag:env,Values=%s", "(^production)|(^dev$)"),
				},
				exclusionFilters: []string{},
				services: []*APIGatewayService{
					devEnvService,
					production,
					newTestService(
						"service-to-be-excluded",
						map[string]string{
							"env": "notdev",
						},
					),
				},
				expectedResult: []*APIGatewayService{
					devEnvService,
					production,
				},
			},
		},
		{
			"multiple filter",
			testData{
				filters: []string{
					fmt.Sprintf("Name=tag:env,Values=%s", "(^dev$)"),
					fmt.Sprintf("Name=tag:service,Values=%s", *devEnvService.restAPI.Name),
				},
				exclusionFilters: []string{},
				services: []*APIGatewayService{
					devEnvService,
					production,
					newTestService(
						"service-to-be-excluded",
						map[string]string{
							"env": "dev",
						},
					),
				},
				expectedResult: []*APIGatewayService{
					devEnvService,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			result := FilterServices(
				tc.testData.services,
				tc.testData.filters,
				tc.testData.exclusionFilters,
			)
			if !reflect.DeepEqual(result, tc.testData.expectedResult) {
				resultStr := ""
				for _, s := range result {
					resultStr += fmt.Sprintf("\n- %s - %v", s.Name(), s.Tags())
				}
				expectedResultStr := ""
				for _, s := range tc.testData.expectedResult {
					expectedResultStr += fmt.Sprintf("\n- %s - %v", s.Name(), s.Tags())
				}

				t.Errorf("[%s] Expected %s,\n got %s", tc.description, expectedResultStr, resultStr)
			}
		})
	}
}
