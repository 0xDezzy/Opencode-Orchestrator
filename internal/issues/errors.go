package issues

import "errors"

var (
	ErrIssueNotFound     = errors.New("linear issue not found")
	ErrIssueArchived     = errors.New("linear issue archived")
	ErrIssueInaccessible = errors.New("linear issue inaccessible")
)

func IsIssueNotFound(err error) bool {
	return errors.Is(err, ErrIssueNotFound)
}

func IsIssueRemoved(err error) bool {
	return errors.Is(err, ErrIssueNotFound) || errors.Is(err, ErrIssueArchived) || errors.Is(err, ErrIssueInaccessible)
}
