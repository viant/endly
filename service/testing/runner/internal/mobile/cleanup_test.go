package mobile

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCleanupStackIsLIFOAndContinuesAfterErrors(t *testing.T) {
	stack := &CleanupStack{}
	order := []string{}
	stack.Push("first", func(context.Context) error {
		order = append(order, "first")
		return nil
	})
	stack.Push("second", func(context.Context) error {
		order = append(order, "second")
		return errors.New("second failed")
	})
	stack.Push("third", func(context.Context) error {
		order = append(order, "third")
		return nil
	})
	errors := stack.Close(context.Background())
	if !reflect.DeepEqual(order, []string{"third", "second", "first"}) {
		t.Fatalf("unexpected cleanup order: %v", order)
	}
	if len(errors) != 1 || errors[0].Name != "second" {
		t.Fatalf("unexpected cleanup errors: %+v", errors)
	}
	if again := stack.Close(context.Background()); !reflect.DeepEqual(again, errors) {
		t.Fatalf("idempotent close changed errors: %+v", again)
	}
}
