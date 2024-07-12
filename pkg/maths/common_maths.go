package maths

// Pretty much transcribed from https://stackoverflow.com/a/147539
// LcmMultiple returns the greatest common multiple of a and b...

func Gcd(a, b int) int {
	// Return greatest common divisor using Euclid's Algorithm.
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func Lcm(a, b int) int {
	// Return lowest common multiple.
	return a * b / Gcd(a, b)
}

func LcmMultiple(args ...int) int {
	// """Return lcm of args."""
	if len(args) == 0 {
		return 1
	}
	rv := args[0]
	for i := 1; i < len(args); i++ {
		rv = Lcm(rv, args[i])
	}
	return rv
}

func CartesianProduct(args ...[]map[string]interface{}) []map[string]interface{} {
	// """Return the Cartesian product of args."""
	if len(args) == 0 {
		return []map[string]interface{}{}
	}
	rv := []map[string]interface{}{}
	rv = append(rv, args[0]...)
	for i := 1; i < len(args); i++ {
		newRV := []map[string]interface{}{}
		for _, row := range args[i] {
			for _, existingRow := range rv {
				newRow := map[string]interface{}{}
				for k, v := range existingRow {
					newRow[k] = v
				}
				for k, v := range row {
					newRow[k] = v
				}
				newRV = append(newRV, newRow)
			}
		}
		rv = newRV
	}
	return rv
}
