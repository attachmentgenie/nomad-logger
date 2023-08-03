build:
	goreleaser build --id $(shell go env GOOS) --single-target --snapshot --clean
darwin:
	goreleaser build --id darwin --snapshot --clean
linux:
	goreleaser build --id linux --snapshot --cleant
snapshot:
	goreleaser release --snapshot --clean
tag:
	git tag $(shell svu next)
	git push --tags
release: tag
	goreleaser --clean
