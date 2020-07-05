package bandit

import "fmt"

type (
	checkable interface {
		Check()
	}
)

func check(ob checkable, label string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println(">>> ERROR CHECKING", label)
			panic(e)
		}
	}()
	ob.Check()
}

func (t *Tree) Check() {
	// Check nodes
	if len(t.nodes) == 0 {
		panic("Uninitialized tree")
	}
	// Check free list
	if t.nextfree > 0 {
		n := t.nextfree
		c := t.numfree
		for n > 0 {
			nd := &t.nodes[n]
			n = nd.left
			c -= 1
		}
		if c != 0 {
			fmt.Println("ERROR: Free list was incorrect")
		}
	}

	// Check tree
	t.check(t.root, 0)
}

func (t *Tree) check(a, p uint) {
	n := &t.nodes[a]
	if n.parent != p {
		//t.PrintTree()
		//panic(fmt.Sprintf("incorrect parentage: %d should have %d but has %d", a, p, n.parent))
	}
	if n.left > 0 {
		t.check(n.left, a)
		t.check(n.right, a)
	}
}
