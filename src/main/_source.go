package main

/*
func test1() {
	a, d := 3, 2
	for {
		f := a + d
		g := 5
		a = g - d
		if f <= g {
			f = g + 1
		} else if g >= a {
			return
		}
		d = 2
	}
}*/

func test2() int {
	i, j := 1, 1
	for i%2 != 0 {
		if j < 4 {
			i++
			j++
		}
	}
	return i + j
}
