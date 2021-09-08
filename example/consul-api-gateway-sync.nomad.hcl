job "consul-api-gateway-sync" {
  datacenters = ["dc1"]
  region      = "dc1"

  constraint {
    attribute = "${meta.cluster_class}"
    value = "backend"
  }

  group "consul-api-gateway-sync" {
    task "consul-api-gateway-sync" {
      driver = "docker"

      config {
        image = "mongey/consul-api-gateway-sync:latest"

        args = [
          "-sleep", "300",
          "-tag", "traefik.enable=true",
          "-tag", "traefik.protocol=https",
          "-tag", "traefik.frontend.passHostHeader=false",
          "-tag", "traefik.frontend.entryPoints=http",
          "-tag", "traefik.frontend.rule=Host:{{ .Name }}.example.org;AddPrefix: /{{.Tags.STAGE }}/"
        ]
      }

      env {
        CONSUL_HTTP_ADDR = "http://${attr.unique.network.ip-address}:8500"
      }
    }
  }
}
