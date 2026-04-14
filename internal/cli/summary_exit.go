package cli

import (
	"fmt"

	"photo-tool/internal/domain"
)

// errIfOperationFailures returns a non-nil error when the batch recorded one or more per-file
// failures, so main can exit non-zero while still printing the full summary (scan/import parity).
func errIfOperationFailures(op string, sum domain.OperationSummary) error {
	if sum.Failed > 0 {
		return fmt.Errorf("%s: %d file(s) failed", op, sum.Failed)
	}
	return nil
}
