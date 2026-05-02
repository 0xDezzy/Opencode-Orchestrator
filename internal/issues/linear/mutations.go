package linear

import (
	"fmt"
	"strings"
)

func validateMutationTarget(issueID string) error {
	if strings.TrimSpace(issueID) == "" {
		return fmt.Errorf("linear issue id is required")
	}
	return nil
}

func validateTransitionState(stateName string) error {
	if strings.TrimSpace(stateName) == "" {
		return fmt.Errorf("linear transition state is required")
	}
	return nil
}
