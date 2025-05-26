package util

type Subject struct {
	Role string // depositor | banker | ...
	Name string // username
}

type Object struct {
	Name string // username of resource onwer
}
