# Generator is pinned. Do not float this version — regenerating with a
# different generator produces diff noise that hides real spec changes.
OAPI_CODEGEN := github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0

.PHONY: generate verify-generated test

generate:
	go run $(OAPI_CODEGEN) -config openapi/codegen.yaml openapi/bundled.yaml

# Fails when the committed generated code does not match the committed spec.
verify-generated: generate
	git diff --exit-code --stat internal/gen/gen.go

test:
	go test ./...
