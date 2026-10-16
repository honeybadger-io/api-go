package apiv3

import "syscall"

// errConnRefused is the error a dial to a port nothing listens on wraps. Windows
// reports it as WSAECONNREFUSED; syscall.ECONNREFUSED there is Go's own
// placeholder, which a real dial never returns. The syscall package doesn't
// export the WSA constants, and golang.org/x/sys/windows would be a dependency
// for one number.
const errConnRefused = syscall.Errno(10061) // WSAECONNREFUSED
