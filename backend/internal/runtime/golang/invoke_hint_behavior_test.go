package golang_test

import (
	"testing"

	runtimego "github.com/gogo/goserverless/internal/runtime/golang"
)

func TestInvokeHintCommandRemainsBoundToArtifact(t *testing.T) {
	impl := runtimego.New(t.TempDir())
	firstArtifact := "/artifacts/daily-report/v3"
	secondArtifact := "/artifacts/billing-sync/v8"

	first := impl.InvokeHint(firstArtifact)
	second := impl.InvokeHint(secondArtifact)

	if len(first.Command) != 1 || first.Command[0] != firstArtifact+"/handler" {
		t.Fatalf(
			"first hint changed after creating a second hint: workdir=%q command=%q",
			first.WorkingDir,
			first.Command,
		)
	}
	if len(second.Command) != 1 || second.Command[0] != secondArtifact+"/handler" {
		t.Fatalf("unexpected second hint command: %q", second.Command)
	}
}
