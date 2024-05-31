module github.com/vividvilla/simplesessions/examples/nethttp-secure-cookie

go 1.18

require (
	github.com/vividvilla/simplesessions/stores/securecookie/v3 v3.0.0
	github.com/vividvilla/simplesessions/v3 v3.0.0
)

require github.com/gorilla/securecookie v1.1.2 // indirect

replace (
	github.com/vividvilla/simplesessions/stores/securecookie/v3 => ../../stores/securecookie
	github.com/vividvilla/simplesessions/v3 => ../..
)
