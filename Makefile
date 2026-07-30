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

.PHONY: generate verify-generated verify-spec update-spec-checksum verify test tools clean-tools

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

# Fails when the vendored spec is not the one the provenance table describes.
#
# The bundle is gitignored upstream — it is a build artifact of `rake
# openapi:bundle`, not a tracked file — so there is no remote blob to diff
# against and the checksum is the only way to answer "is this still the spec we
# think it is". Recording it only in README prose meant nothing checked it; the
# artifact changed twice during one vendoring session, once mid-copy.
#
# Refresh with `make update-spec-checksum` when vendoring deliberately.
verify-spec:
	@actual=`shasum -a 256 openapi/bundled.yaml | awk '{print $$1}'`; \
	expected=`cat openapi/bundled.yaml.sha256`; \
	if [ "$$actual" != "$$expected" ]; then \
		echo "openapi/bundled.yaml does not match its recorded checksum."; \
		echo "  recorded: $$expected"; \
		echo "  actual:   $$actual"; \
		echo "Vendoring a new bundle? Run make update-spec-checksum and update the"; \
		echo "provenance table in openapi/README.md in the same commit."; \
		exit 1; \
	fi; \
	echo "openapi/bundled.yaml matches its recorded checksum"

update-spec-checksum:
	shasum -a 256 openapi/bundled.yaml | awk '{print $$1}' > openapi/bundled.yaml.sha256

# Everything CI should gate on.
verify: verify-spec verify-generated test

test:
	go test ./...
