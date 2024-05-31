module github.com/vividvilla/simplesessions/examples/fasthttp-inmemory

go 1.18

require (
	github.com/valyala/fasthttp v0.0.0-20180901052036-d7688109a57b
	github.com/vividvilla/simplesessions/stores/memory/v3 v3.0.0
	github.com/vividvilla/simplesessions/v3 v3.0.0
)

require (
	github.com/klauspost/compress v1.4.0 // indirect
	github.com/klauspost/cpuid v0.0.0-20180405133222-e7e905edc00e // indirect
	github.com/valyala/bytebufferpool v0.0.0-20160817181652-e746df99fe4a // indirect
)

replace (
	github.com/vividvilla/simplesessions/stores/memory/v3 => ../../stores/memory
	github.com/vividvilla/simplesessions/v3 => ../..
)
