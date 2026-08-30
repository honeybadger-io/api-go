# Extracts each operation's required scope from the bundled OpenAPI spec and
# emits a Go map.
#
# The spec is the only place these live, and hand-copying 114 of them would rot.
# oapi-codegen does not emit security information — v2.8 deliberately stopped
# flattening it, because OpenAPI security can express OR and AND combinations
# that a single string cannot represent.
#
# That is precisely the assumption this script rests on, so it checks it rather
# than trusting it. Honeybadger's v3 grants one bearer scope per operation:
#
#       security:
#       - bearer_auth:
#         - faults:read
#
# Anything else — a second scope, an alternative scheme, an operation with no
# security block that nobody declared unsecured — fails generation.
#
# A silently incomplete map is the outcome worth failing loudly to avoid. The MCP
# server reads a missing operation as "needs no scope", so a dropped entry widens
# what a credential is offered rather than narrowing it. Every operation must
# therefore land in exactly one of two buckets, and the totals reconcile at the
# end:
#
#   operations = scopes emitted + operations explicitly known to be unsecured
BEGIN {
  # Operations that legitimately carry no security block. getToken must stay
  # reachable with any credential so a caller can always discover its own limits.
  unsecured["getToken"] = 1

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

# A new operation closes the previous one.
/^      operationId: / {
  flush()
  op = $2
  seen_security = 0
  scope_count = 0
  scope = ""
  in_security = 0
  expect = 0
  next
}

/^      security:$/ { in_security = 1; next }

in_security && /^      - bearer_auth:$/ { seen_security = 1; expect = 1; next }

# Any other scheme under security: is a shape this script cannot represent.
in_security && /^      - / {
  fail("operation " op " uses a security scheme other than bearer_auth: " $0)
}

expect && /^        - / {
  scope_count++
  scope = $2
  next
}

# Any other key at the operation's own indentation ends the security block.
/^      [a-z]/ { in_security = 0; expect = 0 }

function flush() {
  if (op == "") return
  ops++

  if (!seen_security) {
    if (!(op in unsecured)) {
      fail("operation " op " has no security block and is not listed as unsecured. " \
           "If that is intentional, add it to the unsecured list in scopes.awk with a reason.")
    }
    unsecured_seen++
    return
  }
  if (scope_count != 1) {
    fail("operation " op " declares " scope_count " scopes; this map holds one per " \
         "operation. OR/AND combinations need a richer type than map[string]string.")
  }
  printf "\t%-30s %s,\n", "\"" op "\":", "\"" scope "\""
  emitted++
}

function fail(message) {
  print "scopes.awk: " message > "/dev/stderr"
  failed = 1
  exit 1
}

END {
  if (failed) exit 1
  flush()
  print "}"

  if (ops != emitted + unsecured_seen) {
    print "scopes.awk: " ops " operations but " emitted " scopes and " unsecured_seen \
          " unsecured — every operation must be accounted for" > "/dev/stderr"
    exit 1
  }
  printf "scopes.awk: %d operations = %d scopes + %d unsecured\n", ops, emitted, unsecured_seen > "/dev/stderr"
}
