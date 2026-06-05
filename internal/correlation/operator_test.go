package correlation

import (
	"strings"
	"testing"
)

func TestOperatorIdentity_nonEmptyInTestProcess(t *testing.T) {
	t.Parallel()

	user := OperatorUser()
	uid := OperatorUID()
	if user == "" && uid == "" {
		t.Skip("no OS user in test environment")
	}
	if user == "" {
		t.Fatal("expected os_user when uid is set")
	}
}

func TestAppendEnv_includesOSUserWhenAvailable(t *testing.T) {
	t.Parallel()

	got := AppendEnv(nil, []string{"A=1"})
	joined := strings.Join(got, "\n")
	if u := OperatorUser(); u != "" && !strings.Contains(joined, EnvOSUser+"="+u) {
		t.Fatalf("missing %s in %v", EnvOSUser, got)
	}
	if uid := OperatorUID(); uid != "" && !strings.Contains(joined, EnvOSUID+"="+uid) {
		t.Fatalf("missing %s in %v", EnvOSUID, got)
	}
}
