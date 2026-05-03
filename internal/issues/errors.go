package issues

import "errors"

var ErrIssueNotFound = errors.New("linear issue not found")

func IsIssueNotFound(err error) bool {
	return errors.Is(err, ErrIssueNotFound)
}
