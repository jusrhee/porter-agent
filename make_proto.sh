protoc -I pkg/logstore/lokistore/proto pkg/logstore/lokistore/proto/*.proto --go_out=pkg/logstore/lokistore/proto --go-grpc_out=pkg/logstore/lokistore/proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative