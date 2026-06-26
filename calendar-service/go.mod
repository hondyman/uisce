module calendar-service

go 1.24.0

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0
	github.com/coreos/go-oidc/v3 v3.17.0
	github.com/crewjam/saml v0.5.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/hasura/go-graphql-client v0.10.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/microsoftgraph/msgraph-sdk-go v1.96.0
	github.com/prometheus/client_golang v1.23.2
	github.com/sendgrid/sendgrid-go v3.16.1+incompatible
	github.com/sirupsen/logrus v1.9.4
	github.com/slack-go/slack v0.17.3
	github.com/stretchr/testify v1.11.1
	github.com/teambition/rrule-go v1.8.2
	github.com/twmb/franz-go v1.20.7
	go.temporal.io/sdk v1.40.0
	golang.org/x/oauth2 v0.35.0
	golang.org/x/time v0.14.0
	google.golang.org/api v0.267.0
)

require (
	cloud.google.com/go/auth v0.18.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/beevik/etree v1.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/googleapis/gax-go/v2 v2.17.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/microsoft/kiota-abstractions-go v1.9.3 // indirect
	github.com/microsoft/kiota-authentication-azure-go v1.3.1 // indirect
	github.com/microsoft/kiota-http-go v1.5.4 // indirect
	github.com/microsoft/kiota-serialization-form-go v1.1.2 // indirect
	github.com/microsoft/kiota-serialization-json-go v1.1.2 // indirect
	github.com/microsoft/kiota-serialization-multipart-go v1.1.2 // indirect
	github.com/microsoft/kiota-serialization-text-go v1.1.3 // indirect
	github.com/microsoftgraph/msgraph-sdk-go-core v1.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nexus-rpc/sdk-go v0.5.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/russellhaering/goxmldsig v1.4.0 // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/std-uritemplate/std-uritemplate/go/v2 v2.0.3 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.12.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.temporal.io/api v1.62.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20 // indirect
	google.golang.org/grpc v1.78.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
