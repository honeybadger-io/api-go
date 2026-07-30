# The generator is pinned in internal/tools/go.mod, as a tool directive with its
# own go.sum — not by version string here.
#
# `go run <module>@<version>` looked equivalent and was not: the version is not
# recorded in any go.sum, so `go mod download` never fetches it, and generation
# reaches the network. Worse, the generator needs Go 1.25 while this library
# targets 1.24, so the request also triggers a toolchain switch, which is itself a
# module lookup:
#
#   go: switching to go >= 1.25.0: module lookup disabled by GOPROXY=off
#
# A separate module keeps that Go 1.25 floor off consumers who merely import the
# client, while making the generator a checksummed dependency. Building it to
# ./bin first — rather than `go tool` in place — is what lets the paths in
# codegen.yaml stay relative to this directory.
OAPI_CODEGEN := bin/oapi-codegen

.PHONY: generate verify-generated test tools clean-tools

# GOWORK=off because the tools module is deliberately outside the workspace: it
# exists to hold a dependency the library must not inherit.
$(OAPI_CODEGEN): internal/tools/go.mod internal/tools/go.sum
	GOWORK=off go -C internal/tools build -o "$(CURDIR)/$(OAPI_CODEGEN)" \
		github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

tools: $(OAPI_CODEGEN)

clean-tools:
	rm -rf bin

generate: $(OAPI_CODEGEN)
	./$(OAPI_CODEGEN) -config openapi/codegen.yaml openapi/bundled.yaml
	@# Via a temporary file: redirecting straight at the target truncates it before
	@# awk runs, so a spec that trips one of the script's invariants would leave a
	@# half-written scope map behind. Generate, then move only on success.
	awk -f openapi/scopes.awk openapi/bundled.yaml > apiv3/scopes_gen.go.tmp
	gofmt -w apiv3/scopes_gen.go.tmp
	mv apiv3/scopes_gen.go.tmp apiv3/scopes_gen.go

# Fails when the committed generated code does not match the committed spec.
verify-generated: generate
	git diff --exit-code --stat internal/gen/gen.go apiv3/scopes_gen.go

test:
	go test ./...
