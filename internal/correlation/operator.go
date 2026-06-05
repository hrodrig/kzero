package correlation

import (
	"os"
	"os/user"
	"strings"
)

// EnvOSUser and EnvOSUID are set on hook, custom, and release script subprocesses.
const (
	EnvOSUser = "KZERO_OS_USER"
	EnvOSUID  = "KZERO_OS_UID"
)

// OperatorUser returns the OS username for the kzero process, or empty when unknown.
func OperatorUser() string {
	if u, ok := operatorAccount(); ok {
		return u.Username
	}
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	return strings.TrimSpace(os.Getenv("USERNAME"))
}

// OperatorUID returns the numeric UID as a string, or empty when unknown (e.g. some Windows setups).
func OperatorUID() string {
	if u, ok := operatorAccount(); ok {
		return u.Uid
	}
	return ""
}

func operatorAccount() (*user.User, bool) {
	u, err := user.Current()
	if err != nil || u == nil {
		return nil, false
	}
	return u, true
}
