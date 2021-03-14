package main

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/aws/aws-sdk-go/service/apigateway"
	consulapi "github.com/hashicorp/consul/api"
)

type APIGatewayService struct {
	region  string
	restAPI *apigateway.RestApi
}

func NewService(restAPI *apigateway.RestApi, region string) *APIGatewayService {
	return &APIGatewayService{
		region:  region,
		restAPI: restAPI,
	}
}

func (a *APIGatewayService) ConsulService(tags []string) *consulapi.CatalogRegistration {
	node := a.Name()
	name := a.Name()

	registration := &consulapi.CatalogRegistration{
		Node: node,
		NodeMeta: map[string]string{
			"external-node":  "true",
			"external-probe": "true",
			"registered-by":  "consul-api-gateway-sync",
		},
		Address: a.Address(),
		Service: &consulapi.AgentService{
			Service: name,
			Tags:    tags,
			Meta:    a.Tags(),
		},
		// Creating a service should not modify the node
		// See https://github.com/hashicorp/terraform-provider-consul/issues/101
		SkipNodeUpdate: true,
	}

	return registration
}

func (a *APIGatewayService) Tags() map[string]string {
	tags := make(map[string]string)
	for k, v := range a.restAPI.Tags {
		tags[k] = *v
	}
	return tags
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

	return result
}

func (a *APIGatewayService) Name() string {
	nameWithStage := *(a.restAPI.Name)

	return strings.Replace(nameWithStage, fmt.Sprintf("%s-", a.Stage()), "", 1)
}

func (a *APIGatewayService) Address() string {
	return fmt.Sprintf("%s.execute-api.%s.amazonaws.com", *a.restAPI.Id, a.region)
}

func (a *APIGatewayService) URL() string {
	return fmt.Sprintf("https://%s", a.Address())
}
