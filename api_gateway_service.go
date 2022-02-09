package main

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/service/apigateway"
	consulapi "github.com/hashicorp/consul/api"
)

type APIGatewayService struct {
	region  string
	restAPI *apigateway.RestApi
}

// Tags returns the tags for an APIGatewayService
func (a *APIGatewayService) Tags() map[string]string {
	tags := make(map[string]string)
	for k, v := range a.restAPI.Tags {
		tags[validTag(k)] = validTag(*v)
	}
	return tags
}

// validTag removes non-alphanumeric characters from a tag
// https://github.com/hashicorp/consul/issues/8127
func validTag(s string) string {
	replacementSeperator := "-"
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return ""
	}
	return reg.ReplaceAllString(s, replacementSeperator)
}

// NewService creates a APIGatewayService from a RestAPI
func NewService(restAPI *apigateway.RestApi, region string) *APIGatewayService {
	return &APIGatewayService{
		region:  region,
		restAPI: restAPI,
	}
}

const applicationName = "consul-api-gateway-sync"

// ConsulService builds a consul service
func (a *APIGatewayService) ConsulService(tags []string) *consulapi.CatalogRegistration {
	node := a.ID()
	name := a.Name()

	serviceMeta := a.Tags()
	serviceMeta["external-source"] = "aws"
	registration := &consulapi.CatalogRegistration{
		Node: node,
		NodeMeta: map[string]string{
			"external-node":  "true",
			"external-probe": "true",
			"registered-by":  applicationName,
		},
		Address: a.Address(),
		Service: &consulapi.AgentService{
			Service: name,
			Tags:    tags,
			Port:    443,
			Meta:    serviceMeta,
		},
		Checks: []*consulapi.HealthCheck{
			{
				CheckID:     fmt.Sprintf("service:%s", name),
				Name:        name,
				Node:        node,
				Notes:       "created by consul-api-gateway-sync",
				ServiceName: name,
				Status:      "passing",
			},
		},
		// Creating a service should not modify the node
		// See https://github.com/hashicorp/terraform-provider-consul/issues/101
		SkipNodeUpdate: true,
	}

	return registration
}

func (a *APIGatewayService) Stage() string {
	stage := a.restAPI.Tags["STAGE"]
	if stage == nil {
		return ""
	}
	return *stage
}

type TemplateContext struct {
	Name string
	Tags map[string]string
}

func (a *APIGatewayService) TagsFromTemplate(templates []string) []string {
	result := make([]string, len(templates))
	for i, tmpl := range templates {
		t := template.New(fmt.Sprintf("Template %d", i))

		tt, err := t.Parse(tmpl)
		if err != nil {
			fmt.Println(err)
			continue
		}

		var r bytes.Buffer

		f := TemplateContext{
			Name: a.Name(),
			Tags: a.Tags(),
		}

		if err := tt.Execute(&r, f); err != nil {
			fmt.Println(err)
			continue
		}

		result[i] = r.String()
	}

	return nonEmpty(result)
}

// nonEmpty returns all strings in the slice are not empty
func nonEmpty(s []string) []string {
	result := make([]string, 0)
	for _, v := range s {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func (a *APIGatewayService) ID() string {
	return *a.restAPI.Id
}

func (a *APIGatewayService) Name() string {
	name := *(a.restAPI.Name)
	if a.Stage() != "" {
		name = strings.Replace(name, fmt.Sprintf("%s-", a.Stage()), "", 1)
		name = strings.Replace(name, fmt.Sprintf("-%s", a.Stage()), "", 1)
	}

	serviceName := a.restAPI.Tags["service"]
	if serviceName != nil {
		return *serviceName
	}

	return name
}

func (a *APIGatewayService) Address() string {
	return fmt.Sprintf("%s.execute-api.%s.amazonaws.com", *a.restAPI.Id, a.region)
}

func (a *APIGatewayService) URL() string {
	return fmt.Sprintf("https://%s", a.Address())
}
