all: app.js app.js.map

app.js: *.go
	GOPATH=$${HOME}/src/gamedev/ go1.12.16 get -u golang.org/x/net/websocket
	GOPATH=$${HOME}/src/gamedev/ go1.12.16 get -u github.com/gopherjs/websocket
	GOPHERJS_GOROOT=$$(go1.12.16 env GOROOT) GOPATH=$${HOME}/src/gamedev/ gopherjs build

app.js.map: app.js

clean:
	rm -f app.js app.js.map

.PHONY: all clean
