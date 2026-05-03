package cli

import "testing"

func TestRootCommandRegistersReconcile(t *testing.T) {
	cmd := rootCmd()
	reconcile, _, err := cmd.Find([]string{"reconcile"})
	if err != nil {
		t.Fatal(err)
	}
	if reconcile == nil || reconcile.Name() != "reconcile" {
		t.Fatalf("reconcile command was not registered")
	}
	for _, flag := range []string{"issue", "dry-run", "force", "json"} {
		if reconcile.Flags().Lookup(flag) == nil {
			t.Fatalf("--%s flag was not registered", flag)
		}
	}
}
