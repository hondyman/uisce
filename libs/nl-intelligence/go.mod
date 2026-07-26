module github.com/hondyman/uisce/libs/nl-intelligence

go 1.26.3

require (
	github.com/hondyman/uisce/libs/llm v0.0.0
	github.com/jmoiron/sqlx v1.4.0
)

replace github.com/hondyman/uisce/libs/llm => ../llm
