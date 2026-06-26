module github.com/hondyman/semlayer/backend/cmd/catalog-worker

go 1.24.7

require (
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.11.2
	github.com/segmentio/kafka-go v0.4.49
)

require (
	github.com/hondyman/semlayer/backend v0.0.0
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
)

replace github.com/hondyman/semlayer/backend => ../..
