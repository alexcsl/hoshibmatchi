module github.com/hoshibmatchi/message-service

go 1.25.3

require (
	// ADDED: Dependency on user-service
	github.com/hoshibmatchi/user-service v0.0.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)

// ADDED: Replace directive to find user-service locally
replace github.com/hoshibmatchi/user-service => ../user-service
