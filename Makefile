# Makefile
# Hackliff, 2014-08-19 23:04
# vim:ft=make

default: bin

bin:
	@sh -c "$(CURDIR)/scripts/build.sh"

dev:
	@TF_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

dist:
	@sh -c "$(CURDIR)/scripts/dist.sh $(VERSION)"

updatedeps:
	goop update
	go get -u -v ./...

clean:
	@sh -c "rm bin/*"

.PHONY: default updatedeps dist bin clean
