module server

go 1.25.5

require (
	aidanwoods.dev/go-paseto v1.6.0
	connectrpc.com/connect v1.19.2
	connectrpc.com/validate v0.6.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/caarlos0/env/v11 v11.3.1
	github.com/dhowden/tag v0.0.0-20240417053706-3d75831295e8
	github.com/jmoiron/sqlx v1.4.0
	github.com/joho/godotenv v1.5.1
	github.com/pressly/goose/v3 v3.27.0
	golang.org/x/time v0.15.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.48.0
)

require (
	aidanwoods.dev/go-result v0.3.1 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.9-20250912141014-52f32327d4b0.1 // indirect
	buf.build/go/protovalidate v1.0.0 // indirect
	cel.dev/expr v0.25.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/cel-go v0.26.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/exp v0.0.0-20260218203240-3dfff04db8fa // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260217215200-42d3e9bedb6d // indirect
	modernc.org/libc v1.70.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
