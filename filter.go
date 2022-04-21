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
	if tagValue == nil {
		return false
	}
	v := *tagValue

	match, err := regexp.MatchString(f.values, v)
	if err != nil {
		fmt.Printf("[ERROR] Invalid regexp: %s, %s\n", f.values, err)
		return false
	}
	return match
}

func filterMatches(filter string, service *APIGatewayService) bool {
	f, err := parseFilter(filter)
	if err != nil {
		return false
	}

	res := f.filterMatches(service)
	return res
}

func FilterServices(availableAPIs []*APIGatewayService, filters []string, exclusionFilters []string) []*APIGatewayService {
	filteredServices := make([]*APIGatewayService, 0)
	for _, service := range availableAPIs {
		matchesFilters := true
		for _, filter := range filters {
			if !filterMatches(filter, service) {
				matchesFilters = false
				break
			}
		}
		if matchesFilters {
			filteredServices = append(filteredServices, service)
		}
	}

	finalServices := make([]*APIGatewayService, 0)
	for _, service := range filteredServices {
		shouldBeFiltered := false

		for _, filter := range exclusionFilters {
			if filterMatches(filter, service) {
				shouldBeFiltered = true
			}
		}

		if !shouldBeFiltered {
			finalServices = append(finalServices, service)
		}
	}

	return finalServices
}
