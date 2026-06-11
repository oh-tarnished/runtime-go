// Package system provides Linux system utilities: user queries and power
// management. Most operations require elevated privileges (root or sudo).
package system

import "github.com/oh-tarnished/runtime-go/system/core"

// User is an alias for [core.User], re-exported so callers do not need to
// import the internal core package.
type (
	User = core.User
)
