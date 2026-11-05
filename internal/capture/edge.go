package capture

// pushThrough is how far (pixels) the cursor must be pushed past a screen
// edge before it switches, on platforms where the position is estimated.
const pushThrough = 60

// edgePush accumulates motion that goes past an edge and resets it as soon
// as the cursor moves back.
func edgePush(n, size, over, d float64) float64 {
	switch {
	case n >= size-1 && d > 0, n <= 0 && d < 0:
		return over + abs(d)
	case d != 0:
		return 0
	}
	return over
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
