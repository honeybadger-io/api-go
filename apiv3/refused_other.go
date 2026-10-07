//go:build !windows

package apiv3

import "syscall"

// errConnRefused is the error a dial to a port nothing listens on wraps.
const errConnRefused = syscall.ECONNREFUSED
