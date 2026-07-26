module github.com/hondyman/uisce/libs/saga

go 1.26.3

require (
	github.com/hondyman/uisce/libs/custodian v0.0.0
	github.com/hondyman/uisce/libs/optimizer v0.0.0
)

replace github.com/hondyman/uisce/libs/custodian => ../custodian

replace github.com/hondyman/uisce/libs/optimizer => ../optimizer
