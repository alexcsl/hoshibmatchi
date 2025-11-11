module github.com/hoshibmatchi/api-gateway

go 1.25.3

require (
	github.com/hoshibmatchi/media-service v0.0.0
	github.com/hoshibmatchi/post-service v0.0.0
	github.com/hoshibmatchi/story-service v0.0.0
	github.com/hoshibmatchi/user-service v0.0.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10 // indirect
)

require github.com/golang-jwt/jwt/v5 v5.3.0

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)

// This is the correct way to replace local modules
replace github.com/hoshibmatchi/user-service => ../user-service

replace github.com/hoshibmatchi/post-service => ../post-service

replace github.com/hoshibmatchi/story-service => ../story-service

replace github.com/hoshibmatchi/media-service => ../media-service
