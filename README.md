# consul-api-gateway-sync
A daemon to sync API Gateways from AWS to Consul.

## Service Name
By default the service name will be the name of the API Gateway with the
'STAGE'- tag removed from the start of beginning of the APIGateway's name

## Tags

Pass -tag to set custom tags per service. The value of each tag is evaluated as a go template

### Available Templates
`.Name`
`.Tags`
`.StageNames`

### To register a service with [Traefik][traefik]
* `-tag traefik.enabled=true`
* `-tag traefik.protocol=https`
* `-tag traefik.frontend.passHostHeader=false`
* `-tag traefik.frontend.rule=Host: {{.Name() }}.example.org; AddPrefix: /{{ index .StageNames 0 }}/`

### Required IAM policy


```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "apigateway:GET",
      "Resource": "*"
    }
  ]
}
```

[traefik]: https://traefik.io
