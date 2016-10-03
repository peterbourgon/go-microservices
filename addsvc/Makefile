DOCKER?= docker

build/.done: Dockerfile build/addsvc
	mkdir -p ./build/docker
	cp $^ ./build/docker
	${DOCKER} build -t peterbourgon/addsvc -f build/docker/Dockerfile ./build/docker
	touch $@

build/addsvc:
	GOOS=linux GOARCH=amd64 go build -o $@ ./cmd/addsvc

