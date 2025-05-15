APP_NAME = musannif
ENV = 

default: build-local

run:
	go run cmd/$(APP_NAME)/main.go -serve

build:
	$(ENV) go build -o bin/$(APP_NAME) cmd/$(APP_NAME)/main.go

build-local: build
	./bin/$(APP_NAME) --signup -username username -password password

build-linux-amd64: ENV = GOOS=linux GOARCH=amd64
build-linux-amd64: build

tag:
	python scripts/main.py

clean:
	rm -rf bin/ notes/ || true
	rm musannif.db || true
	rm *.log || true
