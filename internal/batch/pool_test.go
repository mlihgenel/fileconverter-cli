package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

type flakyConverter struct {
	from       string
	to         string
	failBefore int
	attempts   int
}

func (f *flakyConverter) Convert(input string, output string, opts converter.Options) error {
	f.attempts++
	if f.attempts <= f.failBefore {
		return fmt.Errorf("forced failure")
	}
	return os.WriteFile(output, []byte("ok"), 0644)
}

func (f *flakyConverter) SupportsConversion(from, to string) bool {
	return from == f.from && to == f.to
}

func (f *flakyConverter) Name() string {
	return "flaky"
}

func (f *flakyConverter) SupportedConversions() []converter.ConversionPair {
	return []converter.ConversionPair{{From: f.from, To: f.to}}
}

func TestPoolRetryEventuallySucceeds(t *testing.T) {
	from := "utfrom" + strconv.FormatInt(time.Now().UnixNano(), 36)
	to := "utto" + strconv.FormatInt(time.Now().UnixNano()+1, 36)
	fc := &flakyConverter{from: from, to: to, failBefore: 2}
	converter.Register(fc)

	dir := t.TempDir()
	output := filepath.Join(dir, "out."+to)
	jobs := []Job{
		{
			InputPath:  filepath.Join(dir, "in."+from),
			OutputPath: output,
			From:       from,
			To:         to,
		},
	}

	pool := NewPool(1)
	pool.SetRetry(2, 0)
	results := pool.Execute(jobs)
	if len(results) != 1 {
		t.Fatalf("unexpected result count: %d", len(results))
	}

	r := results[0]
	if !r.Success {
		t.Fatalf("expected success, got error: %v", r.Error)
	}
	if r.Attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", r.Attempts)
	}
	if r.OutputSize == 0 {
		t.Fatalf("expected output size to be set")
	}
}

func TestPoolSkippedJobAndSummary(t *testing.T) {
	pool := NewPool(1)
	results := pool.Execute([]Job{
		{InputPath: "a", OutputPath: "b", SkipReason: "output_exists"},
	})
	if len(results) != 1 {
		t.Fatalf("unexpected result count: %d", len(results))
	}
	if !results[0].Skipped {
		t.Fatalf("expected skipped job")
	}

	summary := GetSummary(results, 0)
	if summary.Skipped != 1 {
		t.Fatalf("expected skipped summary to be 1, got %d", summary.Skipped)
	}
	if summary.Failed != 0 {
		t.Fatalf("expected failed summary to be 0, got %d", summary.Failed)
	}
}
