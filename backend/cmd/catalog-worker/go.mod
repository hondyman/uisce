module github.com/hondyman/uisce/backend/cmd/catalog-worker

go 1.25.5

require (
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.11.2
	github.com/segmentio/kafka-go v0.4.49
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
)

require (
	github.com/hondyman/uisce/backend v0.0.0
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
)

replace github.com/hondyman/uisce/backend => ../..
