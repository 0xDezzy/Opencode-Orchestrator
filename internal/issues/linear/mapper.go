package linear

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"issue-orchestrator/internal/issues"
)

func mapAnyIssue(raw any) issues.Issue {
	v := reflect.Indirect(reflect.ValueOf(raw))
	get := func(name string) string {
		f := v.FieldByName(name)
		if !f.IsValid() {
			return ""
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				return ""
			}
			f = f.Elem()
		}
		switch f.Kind() {
		case reflect.String:
			return f.String()
		case reflect.Int, reflect.Int64:
			return fmt.Sprint(f.Int())
		case reflect.Float32, reflect.Float64:
			return fmt.Sprint(f.Float())
		}
		return fmt.Sprint(f.Interface())
	}
	iss := issues.Issue{ID: get("ID"), Identifier: get("Identifier"), Title: get("Title"), Description: get("Description"), URL: get("URL"), Priority: get("Priority"), BranchName: get("BranchName"), Raw: raw}
	if n := get("Number"); n != "" && iss.Identifier == "" {
		iss.Identifier = n
	}
	if f := v.FieldByName("State"); f.IsValid() {
		fv := reflect.Indirect(f)
		if n := fv.FieldByName("Name"); n.IsValid() {
			iss.State = fmt.Sprint(n.Interface())
		}
	}
	if f := v.FieldByName("Team"); f.IsValid() {
		fv := reflect.Indirect(f)
		if k := fv.FieldByName("Key"); k.IsValid() {
			iss.TeamKey = fmt.Sprint(k.Interface())
			if iss.Identifier != "" && iss.TeamKey != "" && iss.Identifier[0] >= '0' && iss.Identifier[0] <= '9' {
				iss.Identifier = iss.TeamKey + "-" + iss.Identifier
			}
		}
	}
	if f := v.FieldByName("Labels"); f.IsValid() {
		iss.Labels = connectionNodeNames(f)
	}
	if f := v.FieldByName("Project"); f.IsValid() {
		iss.ProjectName = projectName(f)
	}
	if f := v.FieldByName("Assignee"); f.IsValid() {
		iss.Assignee = personName(f)
	}
	if iss.Identifier == "" {
		iss.Identifier = iss.ID
	}
	iss.CreatedAt = timeFromField(v, "CreatedAt")
	iss.UpdatedAt = timeFromField(v, "UpdatedAt")
	return iss
}

func eligible(i issues.Issue, o issues.FetchOptions) bool {
	if o.TeamKey != "" && !strings.EqualFold(i.TeamKey, o.TeamKey) {
		return false
	}
	if o.ProjectName != "" && !projectMatches(i.ProjectName, o.ProjectName) {
		return false
	}
	if len(o.ActiveStates) > 0 && !containsFold(o.ActiveStates, i.State) {
		return false
	}
	for _, x := range o.IncludeLabels {
		if !containsFold(i.Labels, x) {
			return false
		}
	}
	for _, x := range o.ExcludeLabels {
		if containsFold(i.Labels, x) {
			return false
		}
	}
	return true
}

func projectMatches(issueProject, configured string) bool {
	issueProject = normalizeProjectRef(issueProject)
	configured = normalizeProjectRef(configured)
	if issueProject == "" || configured == "" {
		return false
	}
	return issueProject == configured || strings.HasSuffix(configured, "-"+issueProject)
}

func normalizeProjectRef(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "-")
	return v
}

func containsFold(xs []string, x string) bool {
	for _, v := range xs {
		if strings.EqualFold(v, x) {
			return true
		}
	}
	return false
}

func connectionNodeNames(v reflect.Value) []string {
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return nil
	}
	nodes := reflect.Indirect(v).FieldByName("Nodes")
	if !nodes.IsValid() || nodes.Kind() != reflect.Slice {
		return nil
	}
	out := make([]string, 0, nodes.Len())
	for i := 0; i < nodes.Len(); i++ {
		node := reflect.Indirect(nodes.Index(i))
		if !node.IsValid() {
			continue
		}
		name := node.FieldByName("Name")
		if name.IsValid() && name.Kind() == reflect.String && name.String() != "" {
			out = append(out, name.String())
		}
	}
	return out
}

func projectName(v reflect.Value) string {
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return ""
	}
	for _, field := range []string{"SlugID", "Name", "ID"} {
		f := v.FieldByName(field)
		if f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
			return f.String()
		}
	}
	return ""
}

func personName(v reflect.Value) string {
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return ""
	}
	for _, field := range []string{"DisplayName", "Name", "Email"} {
		f := v.FieldByName(field)
		if f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
			return f.String()
		}
	}
	return ""
}

func timeFromField(v reflect.Value, name string) time.Time {
	f := v.FieldByName(name)
	if f.IsValid() && f.Type() == reflect.TypeOf(time.Time{}) {
		return f.Interface().(time.Time)
	}
	return time.Now()
}
