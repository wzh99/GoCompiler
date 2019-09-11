package main


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
}

/*
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

func test3(a, x, y int) int {
	z := -a
	if x > 2 {
		a = x * 2
		if y < 5 {
			b := x * 2
			return a + b
		}
	} else if z < 7 {
		return z
	}
	c := x * 2
	return c
}*/
