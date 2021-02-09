module github.com/bruno-anjos/cloud-edge-deployment

go 1.15

require (
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/bruno-anjos/archimedesHTTPClient v0.0.2
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/goccy/go-json v0.4.1
	github.com/golang/geo v0.0.0-20200730024412-e86565bf3f35
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mitchellh/mapstructure v1.3.3
	github.com/nm-morais/demmon-client v1.0.0
	github.com/nm-morais/demmon-common v1.0.0
	github.com/nm-morais/demmon-exporter v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/urfave/cli/v2 v2.2.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	golang.org/x/sys v0.0.0-20201112073958-5cba982894dd // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

replace (
	github.com/bruno-anjos/archimedesHTTPClient v0.0.2 => ../archimedesHTTPClient
	github.com/nm-morais/demmon-client v1.0.0 => ../../nm-morais/demmon-client
	github.com/nm-morais/demmon-common v1.0.0 => ../../nm-morais/demmon-common
	github.com/nm-morais/demmon-exporter v1.0.2 => ../../nm-morais/demmon-exporter
)
