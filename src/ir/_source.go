package ir

func test() int {
	a, d := 3, 2
	for {
		f := a + d
		g := 5
		a := g - d
		if f <= g {
			f := g + 1
		} else if g >= a {
			return g
		}
		d = 2
	}
}
