package capture

import "testing"

func TestEdgePush(t *testing.T) {
	over := 0.0
	// Pushing right at the right edge accumulates.
	for range 4 {
		over = edgePush(1919, 1920, over, 20)
	}
	if over != 80 {
		t.Fatalf("over = %v", over)
	}
	// Moving back resets.
	if over = edgePush(1900, 1920, over, -5); over != 0 {
		t.Fatalf("reset failed: %v", over)
	}
	// Left edge works too; motion inside the screen never accumulates.
	if edgePush(0, 1920, 0, -10) != 10 || edgePush(500, 1920, 0, 10) != 0 {
		t.Fatal("left edge / inside")
	}
}
