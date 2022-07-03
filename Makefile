.PHONY: build
build:
	sam build

build-TodoFunction:
	GOOS=linux CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/main .

.PHONY: init
init: build
	sam deploy --guided

.PHONY: deploy
deploy: build
	sam deploy

.PHONY: delete
delete:
	sam delete
