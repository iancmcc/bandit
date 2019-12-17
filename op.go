package bandit

import (
	"fmt"
	"strings"
)

type operation uint8

const (
	and operation = iota
	or
	xor
)

const (
	LOGGING = false
)

var indent = ""

func enter(s ...interface{}) func() {
	if !LOGGING {
		return func() {}
	}
	fmt.Println(indent, "->", s)
	i := indent
	indent += "  "
	exit := func() {
		fmt.Println(i, "<-", s)
		indent = indent[:len(indent)-2]
	}
	return exit
}

func log(s ...interface{}) {
	if !LOGGING {
		return
	}
	fmt.Println(indent, "  ", s)
}

type (
	node struct {
		prefix uint64
		level  uint
		left   uint
		right  uint
		ul     bool
		incl   bool
	}
	Tree struct {
		root     uint
		nextfree uint
		numfree  uint
		ul       bool
		nodes    []node
	}
)

func treeEquals(at *Tree, bt *Tree, a, b uint) bool {
	an, bn := at.nodes[a], bt.nodes[b]
	if !an.Equals(bn) {
		return false
	}
	if an.left != 0 && !treeEquals(at, bt, an.left, bn.left) {
		return false
	}
	if an.right != 0 && !treeEquals(at, bt, an.right, bn.right) {
		return false
	}
	return true
}

func (n node) Equals(other node) bool {
	return (n.prefix == other.prefix &&
		n.level == other.level &&
		n.ul == other.ul &&
		n.incl == other.incl)
}

func (n node) String() string {
	return fmt.Sprintf(`[ [%d] %b (%d) L:%d R:%d UL:%t INCL:%t ]`,
		n.level, n.prefix, n.prefix, n.left, n.right, n.ul, n.incl)
}

func (t *Tree) node(prefix uint64, level, left, right uint, ul, incl bool) (idx uint) {
	return t.cp(node{
		prefix: prefix,
		level:  level,
		left:   left,
		right:  right,
		ul:     ul,
		incl:   incl,
	})
}

func (t *Tree) cp(n node) (idx uint) {
	defer func() { log("ALLOCATING", idx, n) }()
	if t.numfree > 0 {
		idx = t.nextfree
		t.nextfree = t.nodes[idx].left
		t.nodes[idx] = n
		t.numfree -= 1
		return
	}
	t.nodes = append(t.nodes, n)
	idx = uint(len(t.nodes) - 1)
	return
}

func (t *Tree) takeOwnership(src *Tree, idx uint) (nidx uint) {
	if src == t {
		// We're the owner already
		return idx
	}
	if idx == 0 {
		return 0
	}
	defer func() { log("TAKING OWNERSHIP OF", nidx) }()
	n := src.nodes[idx]
	if n.left > 0 {
		n.left = t.takeOwnership(src, n.left)
	}
	if n.right > 0 {
		n.right = t.takeOwnership(src, n.right)
	}
	return t.cp(n)
}

func (t *Tree) free(src *Tree, idx uint, recursive bool) {
	if src != t {
		// Don't free nodes for a different tree
		return
	}
	if idx == 0 {
		return
	}
	n := src.nodes[idx]
	log("FREEING", idx, n)
	if recursive && n.left > 0 {
		t.free(src, n.left, recursive)
	}
	if recursive && n.right > 0 {
		t.free(src, n.right, recursive)
	}
	t.nodes[idx] = node{left: t.nextfree}
	t.nextfree = idx
	t.numfree += 1
}

func (t *Tree) overlap(at *Tree, a uint, bul bool, op operation) (idx uint) {
	defer enter("OVERLAP", at.nodes[a], bul)()
	defer func() { log("OVERLAP RETURNED", at.nodes[idx]) }()
	if (op == or && bul) || (op == and && !bul) {
		t.free(at, a, true)
		idx = 0
		return
	}
	idx = a
	return
}

func (t *Tree) collision(at, bt *Tree, a, b uint, aul, bul bool, op operation) uint {
	an, bn := at.nodes[a], bt.nodes[b]
	defer enter("COLLISION", a, an, b, bn, aul, bul)()
	var below, includes, above, boundBelow, boundAbove, unbounded bool
	switch op {
	case or:
		below = aul || bul
		includes = (an.incl != aul) || (bn.incl != bul)
		above = (an.ul != aul) || (bn.ul != bul)
	case and:
		below = aul && bul
		includes = (an.incl != aul) && (bn.incl != bul)
		above = (an.ul != aul) && (bn.ul != bul)
	case xor:
		below = aul != bul
		includes = (an.incl != aul) != (bn.incl != bul)
		above = (an.ul != aul) != (bn.ul != bul)
	}
	boundBelow, boundAbove = below != includes, above != includes
	unbounded = boundBelow != boundAbove
	switch {
	case !boundBelow && !boundAbove:
		t.free(at, a, true)
		t.free(bt, b, true)
		return 0
	case boundBelow == an.incl && unbounded == an.ul:
		t.free(bt, b, true)
		return t.takeOwnership(at, a)
	case boundBelow == bn.incl && unbounded == bn.ul:
		t.free(at, a, true)
		return t.takeOwnership(bt, b)
	}
	return t.node(an.prefix, 0, 0, 0, unbounded, boundBelow)
}

func (t *Tree) join(at, bt *Tree, a, b uint, aul, bul bool, op operation) (idx uint) {
	an, bn := at.nodes[a], bt.nodes[b]
	defer enter("JOIN", a, an, b, bn)()
	defer func() { log("JOIN RETURNED", idx, t.nodes[idx]) }()
	level := BranchingBit(an.prefix, bn.prefix)
	prefix := MaskAbove(an.prefix, level)
	var (
		left, right uint
		lt, rt      *Tree
	)
	if ZeroAt(an.prefix, level) {
		lul := aul != an.ul
		left, lt = t.overlap(at, a, bul, op), at
		right, rt = t.overlap(bt, b, lul, op), bt
	} else {
		rul := bul != bn.ul
		left, lt = t.overlap(bt, b, aul, op), bt
		right, rt = t.overlap(at, a, rul, op), at
	}
	if left == 0 {
		idx = t.takeOwnership(rt, right)
	} else if right == 0 {
		idx = t.takeOwnership(lt, left)
	} else {
		idx = t.node(prefix, level, t.takeOwnership(lt, left), t.takeOwnership(rt, right), (&t.nodes[left]).ul != (&t.nodes[right]).ul, false)
	}
	return
}

func (t *Tree) merge(at, bt *Tree, a, b uint, aul, bul bool, op operation) (idx uint) {
	//defer enter("MERGE", a, at.nodes[a], b, bt.nodes[b], aul, bul)()
	//defer func() { log("MERGE RETURNED", idx, t.nodes[idx]) }()
	if a == 0 && b == 0 {
		idx = 0
		return
	}
	if a == 0 {
		idx = t.takeOwnership(bt, t.overlap(bt, b, aul, op))
		return
	}
	if b == 0 {
		idx = t.takeOwnership(at, t.overlap(at, a, bul, op))
		return
	}
	an, bn := at.nodes[a], bt.nodes[b]
	switch {
	case an.level > bn.level:
		log("branch 1")
		if !IsPrefixAt(bn.prefix, an.prefix, an.level) {
			// disjoint trees; encompass
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		// b is somewhere under a
		var (
			left, right uint
			lt, rt      *Tree
		)
		// Won't be needing a again
		t.free(at, a, false)
		if ZeroAt(bn.prefix, an.level) {
			rul := bul != bn.ul
			// b is under the left side of a
			left, lt = t.merge(at, bt, an.left, b, aul, bul, op), t
			right, rt = t.overlap(at, an.right, rul, op), at
		} else {
			// b is under the right side of a
			rul := aul != (&t.nodes[an.left]).ul
			left, lt = t.overlap(at, an.left, bul, op), at
			right, rt = t.merge(at, bt, an.right, b, rul, bul, op), t
		}
		if left == 0 {
			idx = t.takeOwnership(rt, right)
		} else if right == 0 {
			idx = t.takeOwnership(lt, left)
		} else {
			idx = t.node(an.prefix, an.level, t.takeOwnership(lt, left),
				t.takeOwnership(rt, right),
				(&t.nodes[left]).ul != (&t.nodes[right]).ul,
				false)
		}
		return
	case bn.level > an.level:
		log("branch 2")
		if !IsPrefixAt(an.prefix, bn.prefix, bn.level) {
			// disjoint trees; encompass
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		// a is somewhere under b
		var (
			left, right uint
			lt, rt      *Tree
		)
		t.free(bt, b, false)
		if ZeroAt(an.prefix, bn.level) {
			// a is under the left side of b
			lul := aul != an.ul
			left, lt = t.merge(at, bt, a, bn.left, aul, bul, op), t
			right, rt = t.overlap(bt, bn.right, lul, op), bt
		} else {
			rul := bul != (&t.nodes[bn.right]).ul
			left, lt = t.overlap(bt, bn.left, aul, op), bt
			right, rt = t.merge(at, bt, a, bn.right, aul, rul, op), t
		}
		if left == 0 {
			log("TAKING RIGHT OWNERSHIP", right)
			idx = t.takeOwnership(rt, right)
		} else if right == 0 {
			log("TAKING LEFT OWNERSHIP", right)
			idx = t.takeOwnership(lt, left)
		} else {
			log("NEW PARENT NODE")
			idx = t.node(bn.prefix, bn.level, t.takeOwnership(lt, left),
				t.takeOwnership(rt, right),
				(&t.nodes[left]).ul != (&t.nodes[right]).ul,
				false)
		}
		return
	default: // equal level
		log("branch 3")
		if an.prefix != bn.prefix {
			// disjoint trees; encompass
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		if an.level == 0 {
			// Two representations of the same leaf
			idx = t.collision(at, bt, a, b, aul, bul, op)
			return
		}
		// Two internal nodes with same prefix; merge left with left, right with right
		lul := aul != (&t.nodes[an.left]).ul
		rul := bul != (&t.nodes[bn.left]).ul
		left := t.merge(at, bt, an.left, bn.left, aul, bul, op)
		right := t.merge(at, bt, an.right, bn.right, lul, rul, op)
		// Merge takes ownership, so need to try again here
		if left == 0 {
			idx = right
			return
		}
		if right == 0 {
			idx = left
			return
		}
		newul := (t.nodes[left]).ul != (t.nodes[right]).ul
		idx = t.node(an.prefix, an.level, left, right, newul, false)
		return
	}
}

func (t *Tree) Reset() {
	t.root = 0
	t.nodes = t.nodes[:1]
	t.ul = false
	t.numfree = 0
	t.nextfree = 0
}

func (t *Tree) mergeRoot(at, bt *Tree, a, b uint, aul, bul bool, op operation) {
	t.root = t.merge(at, bt, a, b, aul, bul, op)
	switch op {
	case and:
		t.ul = aul && bul
	case or:
		t.ul = aul || bul
	case xor:
		t.ul = aul != bul
	}
}

func (t *Tree) String() string {
	s := []string{}
	iterator := NewIntervalIterator(*t)
	for iterator.Next() {
		s = append(s, iterator.Interval().String())
	}
	return strings.Join(s, ", ")
}
