module github.com/vividvilla/simplesessions/examples/nethttp-inmemory

go 1.18

require (
	github.com/vividvilla/simplesessions/stores/memory/v3 v3.0.0
	github.com/vividvilla/simplesessions/v3 v3.0.0
)

replace (
	github.com/vividvilla/simplesessions/stores/memory/v3 => ../../stores/memory
	github.com/vividvilla/simplesessions/v3 => ../..
)
