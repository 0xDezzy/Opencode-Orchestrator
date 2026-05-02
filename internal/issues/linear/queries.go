package linear

import "reflect"

func reflectNodes(x any) []any {
	v := reflect.Indirect(reflect.ValueOf(x))
	n := v.FieldByName("Nodes")
	if !n.IsValid() {
		return nil
	}
	out := make([]any, 0, n.Len())
	for i := 0; i < n.Len(); i++ {
		out = append(out, n.Index(i).Interface())
	}
	return out
}

type stateRef struct{ id, name string }

func mapState(raw any) stateRef {
	v := reflect.Indirect(reflect.ValueOf(raw))
	s := stateRef{}
	if f := v.FieldByName("ID"); f.IsValid() {
		s.id = f.String()
	}
	if f := v.FieldByName("Name"); f.IsValid() {
		s.name = f.String()
	}
	return s
}
