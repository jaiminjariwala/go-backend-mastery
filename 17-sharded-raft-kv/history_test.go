package main

import (
	"testing"
	"time"

	"github.com/anishathalye/porcupine"
)

func TestPorcupineRejectsStaleRead(t *testing.T) {
	history := []porcupine.Operation{
		{Input: command{Operation: "write", RequestID: "one", Key: "x", Value: "40"}, Call: 1, Return: 2, Output: receipt{Operation: "write", RequestID: "one", Key: "x", Value: "40", Committed: true}},
		{Input: command{Operation: "read", Key: "x"}, Call: 3, Return: 4, Output: receipt{Operation: "read", Key: "x", Value: "old", Found: true, Committed: true}},
	}
	result, _ := porcupine.CheckOperationsVerbose(kvModel, history, time.Second)
	if result != porcupine.Illegal {
		t.Fatalf("checker accepted deliberately stale read: %v", result)
	}
}
