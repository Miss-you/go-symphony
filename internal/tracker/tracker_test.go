package tracker

import (
	"context"
	"reflect"
	"testing"

	"github.com/Miss-you/go-symphony/internal/domain"
)

func TestTrackerReaderContract(t *testing.T) {
	readerType := reflect.TypeOf((*TrackerReader)(nil)).Elem()

	if readerType.Kind() != reflect.Interface {
		t.Fatalf("TrackerReader kind = %v, want interface", readerType.Kind())
	}

	wantMethods := []struct {
		name string
		args []reflect.Type
	}{
		{
			name: "ListCandidates",
			args: []reflect.Type{
				reflect.TypeOf((*context.Context)(nil)).Elem(),
			},
		},
		{
			name: "ListByStates",
			args: []reflect.Type{
				reflect.TypeOf((*context.Context)(nil)).Elem(),
				reflect.TypeOf([]string{}),
			},
		},
		{
			name: "RefreshByIDs",
			args: []reflect.Type{
				reflect.TypeOf((*context.Context)(nil)).Elem(),
				reflect.TypeOf([]string{}),
			},
		},
	}

	if got := readerType.NumMethod(); got != len(wantMethods) {
		t.Fatalf("TrackerReader method count = %d, want %d", got, len(wantMethods))
	}

	workItemsType := reflect.TypeOf([]domain.WorkItem{})
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	for _, want := range wantMethods {
		method, ok := readerType.MethodByName(want.name)
		if !ok {
			t.Fatalf("TrackerReader missing method %q", want.name)
		}
		if got := method.Type.NumIn(); got != len(want.args) {
			t.Fatalf("%s arg count = %d, want %d", want.name, got, len(want.args))
		}
		for i, arg := range want.args {
			if got := method.Type.In(i); got != arg {
				t.Fatalf("%s arg %d = %v, want %v", want.name, i, got, arg)
			}
		}
		if got := method.Type.NumOut(); got != 2 {
			t.Fatalf("%s return count = %d, want 2", want.name, got)
		}
		if got := method.Type.Out(0); got != workItemsType {
			t.Fatalf("%s first return = %v, want %v", want.name, got, workItemsType)
		}
		if got := method.Type.Out(1); got != errorType {
			t.Fatalf("%s second return = %v, want %v", want.name, got, errorType)
		}
	}
}
