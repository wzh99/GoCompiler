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
}*/
/*
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

func test4() int {
	var d, e, f int
	b := 2
	for i := 1; i <= 100; {
		a := b + 1
		c := 2
		if i%2 == 0 {
			d = a + d
			e = 1 + d
		} else {
			d = -c
			f = 1 + a
		}
		i++
		if a < 2 {
			break
		}
	}
	return d + e + f
}
