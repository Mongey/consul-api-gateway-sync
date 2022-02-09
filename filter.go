package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Filter struct {
	name   string
	values string

	original string
}

func parseFilter(input string) (*Filter, error) {
	filter := &Filter{
		original: input,
	}
	foo := strings.Split(input, ",")
	if len(foo) != 2 {
		return nil, fmt.Errorf("invalid filter %s expected to be in format Name=tag:Key,Values=value", input)
	}

	splitName := strings.Split(foo[0], ":")
	if len(splitName) != 2 {
		return nil, fmt.Errorf("invalid filter %s expected to be in format Name=tag:Key,Values=value", input)

	}

	filter.name = splitName[1]

	splitValues := strings.Split(foo[1], "=")
	if len(splitValues) != 2 {
		return nil, fmt.Errorf("invalid filter %s expected to be in format Name=tag:Key,Values=value", input)
	}

	filter.values = splitValues[1]
	return filter, nil
}

func (f *Filter) filterMatches(service *APIGatewayService) bool {
	tagValue := service.restAPI.Tags[f.name]
	if tagValue != nil {
		v := *tagValue

		match, err := regexp.MatchString(f.values, v)
		if err != nil {
			return false
		}
		return match
	}

	return false
}

func filterMatches(filter string, service *APIGatewayService) bool {
	f, err := parseFilter(filter)
	if err != nil {
		return false
	}

	return f.filterMatches(service)
}

func FilterServices(availableAPIs []*APIGatewayService, filters []string) []*APIGatewayService {
	if len(filters) == 0 {
		return availableAPIs
	}
	filteredServices := make([]*APIGatewayService, 0)
	for _, service := range availableAPIs {
		for _, filter := range filters {
			if filterMatches(filter, service) {
				filteredServices = append(filteredServices, service)
			}
		}
	}
	return filteredServices
}
