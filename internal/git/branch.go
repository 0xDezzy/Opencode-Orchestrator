package git

import "strings"

func BranchName(prefix, issueIdentifier, title string) string {
	slug := SafeIssueSlug(issueIdentifier, title)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return slug
	}
	return prefix + "/" + slug
}
