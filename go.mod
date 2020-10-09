module github.com/bruno-anjos/cloud-edge-deployment

go 1.15

require (
	github.com/bruno-anjos/archimedesHTTPClient v0.0.2
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.8.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/urfave/cli/v2 v2.2.0
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

replace (
	github.com/bruno-anjos/archimedesHTTPClient latest => ../archimedesHTTPClient
)