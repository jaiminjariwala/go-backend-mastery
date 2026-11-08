package main

import (
	"fmt"
	"github.com/anishathalye/porcupine"
	"reflect"
)

// modelState describes one group’s sequential KV behavior for Porcupine.
// It does not model cross-shard transactions or unfinished client operations.
type modelState struct {
	KV       map[string]string
	Requests map[string]command
}

// A whole-group model (not partitioned by key): request IDs span that group, and retrying
// an older request must not restore its value after a newer write has committed.
var kvModel = porcupine.Model{
	Init:  func() interface{} { return modelState{map[string]string{}, map[string]command{}} },
	Equal: func(a, b interface{}) bool { return reflect.DeepEqual(a, b) },
	Step: func(state, input, output interface{}) (bool, interface{}) {
		old := state.(modelState)
		in := input.(command)
		out := output.(receipt)
		if !out.Committed || out.Key != in.Key || out.Operation != in.Operation {
			return false, state
		}
		if in.Operation == "read" {
			value, found := old.KV[in.Key]
			return value == out.Value && found == out.Found, state
		}
		if out.RequestID != in.RequestID || out.Value != in.Value {
			return false, state
		}
		if previous, ok := old.Requests[in.RequestID]; ok {
			return previous == in, state
		}
		next := modelState{map[string]string{}, map[string]command{}}
		for k, v := range old.KV {
			next.KV[k] = v
		}
		for k, v := range old.Requests {
			next.Requests[k] = v
		}
		if in.Operation == "delete" {
			delete(next.KV, in.Key)
		} else {
			next.KV[in.Key] = in.Value
		}
		next.Requests[in.RequestID] = in
		return true, next
	},
	DescribeOperation: func(input, output interface{}) string {
		in := input.(command)
		out := output.(receipt)
		return fmt.Sprintf("%s %s=%q → %q (id %s)", in.Operation, in.Key, in.Value, out.Value, in.RequestID)
	},
	DescribeState: func(state interface{}) string { return fmt.Sprint(state.(modelState).KV) },
}
