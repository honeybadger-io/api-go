# Extracts each operation's required scope from the bundled OpenAPI spec and
# emits a Go map.
#
# The spec is the only place these live, and hand-copying 111 of them would rot.
# oapi-codegen does not emit security information, hence this step.
#
# Every security block in the bundle has the same three-line shape:
#
#       security:
#       - bearer_auth:
#         - faults:read
#
# so the scope is the indented item directly under bearer_auth. An operation with
# no security block requires no scope and is simply absent from the map — today
# that is getToken alone, which must stay reachable so a credential can always
# discover its own limits.
BEGIN {
  print "// Code generated from openapi/bundled.yaml by openapi/scopes.awk. DO NOT EDIT."
  print ""
  print "package apiv3"
  print ""
  print "// OperationScopes maps an operationId to the scope its endpoint requires."
  print "//"
  print "// Absent means no scope is needed. Generated from the spec so it cannot drift"
  print "// from what the API actually enforces."
  print "var OperationScopes = map[string]string{"
}
/^      operationId: / { op = $2; next }
/^      - bearer_auth:$/ { expect = 1; next }
expect && /^        - / { printf "\t%-30s %s,\n", "\"" op "\":", "\"" $2 "\""; expect = 0; next }
{ expect = 0 }
END { print "}" }
