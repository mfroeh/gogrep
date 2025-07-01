package main

import "math"

var quantifierOptional = &quantifier{Min: 0, Max: 1}
var quantifierPlus = &quantifier{Min: 1, Max: math.MaxInt}
var quantifierMul = &quantifier{Min: 0, Max: math.MaxInt}

// {m,n} : {m, n}
// ? : {0, 1}
// * : {0, infty}
// + : {1, infty}
type quantifier struct {
	Min int
	Max int
}

type astNode interface {
	// determines if the node matches the given string
	// the returned number is the maximum length of the match
	//match(s string) int
}

// ()
type group struct {
	Children   []astNode
	Quantifier *quantifier
}

// |
type choices struct {
	Choices [][]astNode
}

// []
type bracket struct {
	Negate     bool
	Chars      []byte
	Ranges     []charRange
	Quantifier *quantifier
}

type charRange struct {
	// a-Z
	From byte
	To   byte
}

type char struct {
	Char       byte
	Quantifier *quantifier
}

func main() {
	println("Hello World!")
}
