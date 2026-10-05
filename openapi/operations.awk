# Extracts each operation's HTTP method from the bundled OpenAPI spec and emits
# a Go map, for apiv3's retry policy.
#
# Whether a failed request may be resent, and how long it may run, depends on
# its method: v3 makes every PATCH, PUT and DELETE converge when sent twice, and
# a POST may create twice. The facade calls generated operations through
# closures that hide the method, and a transport failure has no response to read
# it from, so the method is looked up by operationId. Hand-copying 121 of them
# would rot.
#
# An operation starts at its method line under paths:
#
#       /v3/projects/{project_id}/check_ins/{check_in_id}:
#         patch:
#           operationId: updateCheckIn
#
# Every method block must yield exactly one operationId, and the totals
# reconcile at the end.
BEGIN {
  methods["get"] = "GET"; methods["post"] = "POST"; methods["put"] = "PUT"
  methods["patch"] = "PATCH"; methods["delete"] = "DELETE"

  print "// Code generated from openapi/bundled.yaml by openapi/operations.awk. DO NOT EDIT."
  print ""
  print "package apiv3"
  print ""
  print "// operationMethods maps an operationId to its HTTP method. Generated from the"
  print "// spec so the retry rules cannot drift from it."
  print "var operationMethods = map[string]string{"
}

/^paths:$/ { in_paths = 1; next }

# The next top-level key ends paths:.
in_paths && /^[a-z]/ { flush(); in_paths = 0; next }

!in_paths { next }

# A new path closes the operation before it.
/^  [^ ]/ { flush(); next }

/^    [a-z]+:$/ {
  flush()
  key = $1
  sub(/:$/, "", key)
  if (!(key in methods)) next  # parameters: and the like, shared by the path's methods
  method = methods[key]
  op = ""
  blocks++
  next
}

method != "" && /^      operationId: / {
  if (op != "") fail("method block declares a second operationId: " $2 " after " op)
  op = $2
  next
}

function flush() {
  if (method == "") return
  if (op == "") fail("a " method " operation has no operationId")
  printf "\t%-30s \"%s\",\n", "\"" op "\":", method
  emitted++
  method = ""
  op = ""
}

function fail(message) {
  print "operations.awk: " message > "/dev/stderr"
  failed = 1
  exit 1
}

END {
  if (failed) exit 1
  flush()
  print "}"

  if (blocks != emitted) {
    print "operations.awk: " blocks " method blocks but " emitted " operations emitted" > "/dev/stderr"
    exit 1
  }
  printf "operations.awk: %d operations\n", emitted > "/dev/stderr"
}
