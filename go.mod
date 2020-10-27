module github.com/bruno-anjos/cloud-edge-deployment

go 1.15

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/bruno-anjos/archimedesHTTPClient v0.0.2
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/golang/geo v0.0.0-20200730024412-e86565bf3f35
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.8.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/urfave/cli/v2 v2.2.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

replace github.com/bruno-anjos/archimedesHTTPClient v0.0.2 => ../archimedesHTTPClient
