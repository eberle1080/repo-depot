module github.com/eberle1080/repo-depot/server

go 1.25.1

require (
	github.com/amp-labs/amp-common v0.0.0-20260302204307-622fb6d9020b
	github.com/eberle1080/jsonrpc v0.0.0-20260128005140-00c6f4b1b5c1
	github.com/eberle1080/mcp v0.0.0-20260409063442-1ceebf6ea4c8
	github.com/eberle1080/mcp-protocol v0.0.0-20260128040518-5dfb09d0111d
	github.com/eberle1080/repo-depot/shared v0.0.0-00010101000000-000000000000
	github.com/rabbitmq/amqp091-go v1.10.0
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	facette.io/natsort v0.0.0-20181210072756-2cd4dd1e2dcb // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/neilotoole/slogt v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/viant/afs v1.30.0 // indirect
	github.com/viant/gosh v0.2.4 // indirect
	github.com/viant/scy v0.27.0 // indirect
	github.com/viant/toolbox v0.39.0 // indirect
	github.com/viant/xreflect v0.7.3 // indirect
	github.com/viant/xunsafe v0.10.3 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelslog v0.14.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/log v0.15.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251213004720-97cd9d5aeac2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/eberle1080/repo-depot/shared => ../shared
