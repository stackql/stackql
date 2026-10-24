package intrinsic //nolint:testpackage // Exercises completion policy independently of HTTP fixtures.

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
)

func TestOperationCompletionPollsBeforeReturningResource(t *testing.T) {
	var diagnostics bytes.Buffer
	links := []string{}
	waits := 0
	read := func(_ context.Context, link string, inputs map[string]any) (omnisdk.Row, error) {
		links = append(links, link)
		if inputs["project"] != "p" {
			t.Errorf("scope=%v", inputs)
		}
		if link == "https://original/operation" {
			return omnisdk.Row{"status": "DONE", "targetLink": "https://original/resource"}, nil
		}
		return omnisdk.Row{"name": "completed", "direction": "INGRESS"}, nil
	}
	wait := func(_ context.Context, duration time.Duration) error {
		waits++
		if duration != 10*time.Second {
			t.Errorf("interval=%v", duration)
		}
		return nil
	}
	completion := newLinkedOperationCompletion(read, wait, &diagnostics)
	row, err := completion.Complete(context.Background(), "google.compute.firewalls", "update", map[string]any{"project": "p"}, omnisdk.Row{"kind": "compute#operation", "operationType": "patch", "status": "PENDING", "selfLink": "https://original/operation"})
	if err != nil {
		t.Fatal(err)
	}
	if row["name"] != "completed" || waits != 1 || !reflect.DeepEqual(links, []string{"https://original/operation", "https://original/resource"}) {
		t.Fatalf("row=%v waits=%d links=%v", row, waits, links)
	}
	expected := "compute#operation: patch in progress, 10 seconds elapsed\ncompute#operation: patch complete\n"
	if diagnostics.String() != expected {
		t.Fatalf("diagnostics=%q", diagnostics.String())
	}
}

func TestOperationCompletionCancellationAndDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	completion := newLinkedOperationCompletion(func(context.Context, string, map[string]any) (omnisdk.Row, error) {
		t.Fatal("canceled/deleted operation read its target")
		return nil, context.Canceled
	}, func(context.Context, time.Duration) error { t.Fatal("canceled operation waited"); return nil }, &bytes.Buffer{})
	operation := omnisdk.Row{"kind": "compute#operation", "operationType": "delete", "status": "DONE"}
	if _, err := completion.Complete(ctx, "google.compute.firewalls", "delete", nil, operation); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if row, err := completion.Complete(context.Background(), "google.compute.firewalls", "delete", nil, operation); row != nil || err != nil {
		t.Fatalf("row=%v error=%v", row, err)
	}
}

func TestCompletionLinkMatchingUsesDeclaredPath(t *testing.T) {
	template := "https://catalog.example/compute/v1/projects/{project}/global/operations/{operation}"
	params, ok := completionLinkParams(template, "https://operation.example/compute/v1/projects/p/global/operations/o")
	if !ok || !reflect.DeepEqual(params, map[string]string{"project": "p", "operation": "o"}) {
		t.Fatalf("params=%v matched=%v", params, ok)
	}
	for _, link := range []string{"https://operation.example/unrelated", "https://operation.example/compute/v1/projects/p/global/operations/o?scope=other", "/compute/v1/projects/p/global/operations/o"} {
		if _, matched := completionLinkParams(template, link); matched {
			t.Fatalf("accepted undeclared link %s", link)
		}
	}
}

func TestCompletionLinkCannotChangeExplicitScope(t *testing.T) {
	original := map[string]any{"project": "explicit-project"}
	if err := checkCompletionScope(map[string]string{"project": "other-project"}, original); err == nil {
		t.Fatal("completion changed project")
	}
	if err := checkCompletionScope(map[string]string{"project": "explicit-project", "operation": "new-token"}, original); err != nil {
		t.Fatal(err)
	}
}
