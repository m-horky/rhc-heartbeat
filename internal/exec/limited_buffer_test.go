package exec

import "testing"

// TestLimitedBufferEnforcesPerStreamLimit verifies that each stream has its own output budget.
//
// Given multiple output streams with individual limits, when each writes excess data,
// then each captured stream is independently bounded and marked as truncated.
func TestLimitedBufferEnforcesPerStreamLimit(t *testing.T) {
	t.Parallel()

	stdout := limitedBuffer{limit: 3}
	stderr := limitedBuffer{limit: 3}

	if n, err := stdout.Write([]byte("abcd")); n != 4 || err != nil {
		t.Fatalf("stdout.Write() = (%d, %v), want (4, nil)", n, err)
	}

	if n, err := stderr.Write([]byte("efgh")); n != 4 || err != nil {
		t.Fatalf("stderr.Write() = (%d, %v), want (4, nil)", n, err)
	}

	if stdout.buffer.Len() != 3 || stderr.buffer.Len() != 3 {
		t.Errorf("captured stream sizes = (%d, %d), want (3, 3)", stdout.buffer.Len(), stderr.buffer.Len())
	}

	if !stdout.truncated || !stderr.truncated {
		t.Errorf("truncated flags = (%t, %t), want (true, true)", stdout.truncated, stderr.truncated)
	}
}
