.PHONY: app
app:
	go build -v ./webapp/go/src/app
	cp ./app ./webapp/go/app

.PHONY: run
run: app
	cd ./webapp/go/ && ./app

.PHONY: ensure
ensure:
	cd webapp/go/src/app && dep ensure -vendor-only

.PHONY: update
update:
	cd webapp/go/src/app && dep ensure

.PHONY: test
test:
	go test -v ./webapp/go/src/app

.PHONY: deploy
deploy:
	@./scripts/deploy
