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

func syncGateways(logger hclog.Logger) error {
	var awsRegion string
	var tagFlags = StringSet{}

	flag.StringVar(&awsRegion, "aws-region", "us-west-2", "the aws region to search for api-gateways")
	flag.Var(&tagFlags, "tag", "a template to add as a tag")
	sleepTime := flag.Int("sleep", 10, "the number of seconds to sleep while polling")
	flag.Parse()

	logger.Info("creating api-gateway metadata client")
	mySession := session.Must(session.NewSession())
	svc := apigateway.New(mySession, aws.NewConfig().WithRegion(awsRegion))

	consulClient, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		logger.Error("failed to build consul client", "error", err)
		return err
	}

	logger.Info("starting gateway watch")
	for {
		restAPIResp, err := svc.GetRestApis(&apigateway.GetRestApisInput{})
		if err != nil {
			logger.Error("Failed to list rest apis", "error", err)
			return err
		}

		logger.Debug("Got rest-apis", "items", len(restAPIResp.Items))

		for _, item := range restAPIResp.Items {
			service := NewService(item, awsRegion)
			tags := service.TagsFromTemplate(tagFlags.value)
			logger.Info("Registering service", "service", service.Name(), "address", service.Address(), "tags", tags)

			_, err := consulClient.Catalog().Register(service.ConsulService(tags), nil)
			if err != nil {
				logger.Error("Failed to register", "error", err, "service", service.Tags())
			}
		}

		logger.Info("Sleeping", "time", *sleepTime)
		time.Sleep(time.Second * time.Duration(*sleepTime))
	}

}

func main() {
	logger := hclog.Default()
	err := syncGateways(logger)

	if err != nil {
		logger.Error("Syncing Gateways failed", "error", err)
		os.Exit(1)
	}
}
