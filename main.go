package main

import (
	"flag"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	consulapi "github.com/hashicorp/consul/api"
	hclog "github.com/hashicorp/go-hclog"
)

type TagFlags struct {
	value []string
}

func (f *TagFlags) String() string {
	return ""
}

func (f *TagFlags) Set(s string) error {
	f.value = append(f.value, s)
	return nil
}

var tagFlags = TagFlags{}

func main() {
	var awsRegion string

	flag.Var(&tagFlags, "tag", "a template to add as a tag")
	flag.StringVar(&awsRegion, "aws-region", "us-west-2", "the aws region to search for api-gateways")
	sleepTime := flag.Int("sleep", 10, "the number of seconds to sleep while polling")
	flag.Parse()

	tagTemplates := tagFlags.value

	logger := hclog.Default()
	logger.Info("creating api-gateway metadata client")

	mySession := session.Must(session.NewSession())
	svc := apigateway.New(mySession, aws.NewConfig().WithRegion(awsRegion))

	consulClient, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		logger.Error("failed to build consul client", "error", err)
		os.Exit(1)
	}

	logger.Info("starting gateway watch")

	for {
		restAPIResp, err := svc.GetRestApis(&apigateway.GetRestApisInput{})

		if err != nil {
			logger.Error("Failed to list rest apis", "error", err)
			continue
		}

		for _, item := range restAPIResp.Items {
			service := NewService(item, awsRegion)
			tags := service.TagsFromTemplate(tagTemplates)
			logger.Info("Registering service", "service", service.Name(), "address", service.Address(), "tags", tags)

			_, err := consulClient.Catalog().Register(service.ConsulService(tags), nil)
			if err != nil {
				logger.Error("Failed to register", "error", err)
			}
		}

		logger.Info("Sleeping", "time", *sleepTime)
		time.Sleep(time.Second * time.Duration(*sleepTime))
	}
}
