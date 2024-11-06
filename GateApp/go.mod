module GateApp

go 1.23.0

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/3zheng/railcommon v0.0.1
	github.com/go-sql-driver/mysql v1.8.1 // indirect
)

replace github.com/3zheng/railcommon v0.0.1 => ../../railcommon

require (
	github.com/3zheng/railproto v0.0.3
	google.golang.org/protobuf v1.35.1
)
