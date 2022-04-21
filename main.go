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
	var filterFlags = StringSet{}
	var exclusionFilterFlags = StringSet{}

	flag.StringVar(&awsRegion, "aws-region", "us-west-2", "the aws region to search for api-gateways")
	flag.Var(&filterFlags, "filter", "filters")
	flag.Var(&exclusionFilterFlags, "exclude", "filters")
	flag.Var(&tagFlags, "tag", "a template to add as a tag")
	sleepTime := flag.Int("sleep", 90, "the number of seconds to sleep while polling")
	flag.Parse()

	logger.Info("creating api-gateway metadata client")
	client, err := NewClient(filterFlags.value, exclusionFilterFlags.value, tagFlags.value, awsRegion, logger)
	if err != nil {
		return err
	}

	logger.Info("starting gateway watch")
	for {
		err := client.registerServices()
		if err != nil {
			logger.Error("starting gateway watch")
		}
		logger.Debug("Sleeping", "time", *sleepTime)
		time.Sleep(time.Second * time.Duration(*sleepTime))
	}
}

type Client struct {
	flags                []string
	exclusionFilterFlags []string
	tags                 []string
	region               string
	logger               hclog.Logger
	svc                  *apigateway.APIGateway
	consulClient         *consulapi.Client
}

func NewClient(flags []string, exclusionFilterFlags []string, tags []string, region string, logger hclog.Logger) (*Client, error) {
	mySession := session.Must(session.NewSession())
	svc := apigateway.New(mySession, aws.NewConfig().WithRegion(region))
	consulClient, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		logger.Error("failed to build consul client", "error", err)
		return nil, err
	}

	return &Client{flags, exclusionFilterFlags, tags, region, logger, svc, consulClient}, nil
}

func (c *Client) registerServices() error {
	restAPIResp, err := c.svc.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		c.logger.Error("Failed to list rest apis", "error", err)
		return err
	}

	c.logger.Debug("Got rest-apis", "items", len(restAPIResp.Items))

	services := make([]*APIGatewayService, 0)
	for _, item := range restAPIResp.Items {
		stages, err := c.svc.GetStages(&apigateway.GetStagesInput{RestApiId: item.Id})
		if err != nil {
			c.logger.Error("Failed to list stages", "error", err)
			return err
		}

		stageNames := make([]string, 0)
		if stages.Item != nil {
			for _, stage := range stages.Item {
				stageNames = append(stageNames, *(stage.StageName))
			}
		}

		service := NewService(item, c.region, stageNames)

		c.logger.Info("RestAPI",
			"service_id", service.ID(),
			"StageNames", service.StageNames,
			"service", service.Name(),
			"tags", service.Tags(),
		)
		services = append(services, service)
	}

	filteredServices := FilterServices(services, c.flags, c.exclusionFilterFlags)
	c.logger.Info("Got", len(services), ", ", len(filteredServices), "remain")

	//registedServices, _, err := consulClient.Catalog().Services(&consulapi.QueryOptions{
	//NodeMeta: map[string]string{
	//"registered-by": applicationName,
	//},
	//})

	//servicesToRegister, servicesToRegister, err := services(services, registedServices)

	for _, service := range filteredServices {
		consulService := service.ConsulService(c.tags)
		c.logger.Info("Registering service", "service", service.Name(), "address", service.Address())
		_, err := c.consulClient.Catalog().Register(consulService, nil)
		if err != nil {
			c.logger.Error("Failed to register", "error", err, "service", service.Tags())
		}
	}
	return nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:       "consul-api-gateway",
		Level:      hclog.Info,
		JSONFormat: true,
	})
	err := syncGateways(logger)

	if err != nil {
		logger.Error("Syncing Gateways failed", "error", err)
		os.Exit(1)
	}
}
