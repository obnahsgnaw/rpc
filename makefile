.PHONY: help
help:
	@echo "usage: make <option> <params>"
	@echo "options and effects:"
	@echo "    help   : Show help"
	@echo "version options    :"
	@echo "    major  : Generate major version number"
	@echo "    minor  : Generate minor version number"
	@echo "    version  : Generate auto version number"
	@echo "    changelog: Generate change log file and modify tag"

.PHONY: test
test:
	@echo "Testing..."
	@go test
	@echo "Done"

.PHONY: check
check:
	@echo "Checking..."
	@go fmt ./
	@go vet ./
	@echo "Done"

.PHONY: cover
cover:
	@echo "Covering..."
	@go test -coverprofile cover.out
	@go tool cover -html=cover.out
	@echo "Done"

.PHONY: lint
lint:
	@echo "Linting..."
	golangci-lint run
	@echo "Done"

.PHONY: major
major:
	@echo "Version from ${shell git describe --tags `git rev-list --tags --max-count=1`} to v${shell gsemver bump major}"
	@git tag -a "v${shell gsemver bump major}"

.PHONY: minor
minor:
	@echo "Version from ${shell git describe --tags `git rev-list --tags --max-count=1`} to v${shell gsemver bump minor}"
	@git tag -a "v${shell gsemver bump minor}"

.PHONY: version
version:
	@echo "Version from ${shell git describe --tags `git rev-list --tags --max-count=1`} to v${shell gsemver bump pitch}"
	@git tag -a "v${shell gsemver bump}"

.PHONY: changelog
changelog:
	@echo "Generating change log and tag..."
	@git-chglog -o CHANGELOG.md
	@git add CHANGELOG.md
	@git commit -m "chore(release): ${shell git describe --tags `git rev-list --tags --max-count=1`}"
	@git tag -a -f ${shell git describe --tags `git rev-list --tags --max-count=1`}
	@echo "Done"