auth:
	go build -o auth ./cmd/auth
comments:
	go build -o comments ./cmd/comments
posts:
	go build -o posts ./cmd/posts
search:
	go build -o search ./cmd/search
clean:
	rm auth comments posts ./search
