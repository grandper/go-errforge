package nofmterrorf

// A dot import and a single-line import declaration: the errforge import is
// added as its own declaration, and fmt stays because Println is still used.

import . "fmt"

func dotted() error {
	return Errorf("an error occurred") // want `do not use fmt.Errorf: use errforge\.New$`
}

func dottedPrint() {
	Println("still needs fmt")
}
