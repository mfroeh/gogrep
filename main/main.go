package main

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

type root struct {
	children astNode
}

// ()
type group struct {
	Children   []astNode
	Quantifier *quantifier
}

// |
type choice struct {
	choices [][]astNode
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
