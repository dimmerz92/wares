package sessions_test

import (
	"reflect"
	"testing"

	"github.com/dimmerz92/quicky/sessions"
)

func TestGobEncoder(t *testing.T) {
	type testStruct struct {
		String string
		Int    int
		Float  float64
		Bool   bool
	}

	tests := []struct {
		name         string
		value        any
		receiver     any
		marshalErr   bool
		unmarshalErr bool
		panic        bool
	}{
		{name: "nil", value: nil, marshalErr: true},
		{name: "chan", value: make(chan struct{}), marshalErr: true},
		{name: "string", value: "test", receiver: new(string)},
		{name: "int", value: 123, receiver: new(int)},
		{name: "float", value: 123.456, receiver: new(float64)},
		{name: "bool", value: true, receiver: new(bool)},
		{name: "map[string]any", value: map[string]any{"test": 123}, receiver: &map[string]any{}},
		{name: "struct", value: testStruct{String: "test", Int: 123, Float: 123.456, Bool: true}, receiver: &testStruct{}},
	}

	e := sessions.NewGobEncoder()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if test.panic && r == nil {
					t.Error("expected panic")
				}
				if !test.panic && r != nil {
					t.Errorf("unexpected panic: %v", r)
				}
			}()

			encoded, err := e.Marshal(test.value)
			if test.marshalErr {
				if err == nil {
					t.Fatal("expected marshal error")
				}
				return
			}
			if !test.marshalErr && err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			err = e.Unmarshal(encoded, test.receiver)
			if test.unmarshalErr {
				if err == nil {
					t.Fatal("expected unmarshal error")
				}
				return
			}
			if !test.unmarshalErr && err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if !reflect.DeepEqual(reflect.ValueOf(test.receiver).Elem().Interface(), test.value) {
				t.Errorf("expected %#v, got %#v", test.value, test.receiver)
			}
		})
	}
}
