package issues

import (
	"errors"
	"testing"
)

func TestRemovedIssueErrorClassification(t *testing.T) {
	for _, err := range []error{ErrIssueNotFound, ErrIssueArchived, ErrIssueInaccessible} {
		if !IsIssueRemoved(err) {
			t.Fatalf("%v was not classified as removed", err)
		}
	}
	if IsIssueRemoved(errors.New("temporary failure")) {
		t.Fatal("temporary failure was classified as removed")
	}
}
