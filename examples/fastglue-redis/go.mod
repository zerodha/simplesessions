module github.com/vividvilla/simplesessions/examples/fastglue-redis

go 1.18

require (
	github.com/redis/go-redis/v9 v9.5.1
	github.com/valyala/fasthttp v1.54.0
	github.com/vividvilla/simplesessions/stores/redis/v3 v3.0.0
	github.com/vividvilla/simplesessions/v3 v3.0.0
	github.com/zerodha/fastglue v1.8.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fasthttp/router v1.5.0 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/savsgio/gotils v0.0.0-20240303185622-093b76447511 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/vividvilla/simplesessions/stores/redis/v3 => ../../stores/redis
	github.com/vividvilla/simplesessions/v3 => ../..
)
