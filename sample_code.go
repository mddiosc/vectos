package main

import "fmt"

func main() {
	fmt.Println("Hola Mundo")
}

func suma(a, b int) int {
	return a + b
}

func resta(a, b int) int {
	return a - b
}

func multiplicacion(a, b int) int {
	return a * b
}

func division(a, b, c int) int {
	if b == 0 || c == 0 {
		return 0
	}
	return a / b / c
}
