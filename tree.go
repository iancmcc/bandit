package bandit

import (
	"fmt"
	"strings"
)

type (
	operation uint8
	node      struct {
		prefix uint64
		level  uint
		parent uint
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

const (
	and operation = iota
	or
	xor
	common
)

func treeEquals(at *Tree, bt *Tree, a, b uint) bool {
	an, bn := &at.nodes[a], &bt.nodes[b]
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

func (n *node) Equals(other *node) bool {
	return (n.prefix == other.prefix &&
		n.level == other.level &&
		n.ul == other.ul &&
		n.incl == other.incl)
}

func (n *node) String() string {
	return fmt.Sprintf(`[ [%d] %b (%d) L:%d R:%d UL:%t INCL:%t ]`,
		n.level, n.prefix, n.prefix, n.left, n.right, n.ul, n.incl)
}

func (n *node) boundBoth() bool {
	return n.incl && !n.ul
}

func (n *node) boundBelow() bool {
	return n.incl && n.ul
}

func (n *node) boundAbove() bool {
	return !n.incl && n.ul
}

func (t *Tree) node(prefix uint64, level, left, right uint, ul, incl bool) (idx uint) {
	return t.cp(&node{
		prefix: prefix,
		level:  level,
		left:   left,
		right:  right,
		ul:     ul,
		incl:   incl,
	})
}

func (t *Tree) fix(idx uint) {
	if idx == 0 {
		return
	}
	n := (&t.nodes[idx])
	if n.level == 0 {
		return
	}
}

func (t *Tree) cp(n *node) (idx uint) {
	if t.numfree > 0 {
		idx = t.nextfree
		t.nextfree = (&t.nodes[idx]).left
		t.nodes[idx] = *n
		t.numfree -= 1
	} else {
		t.nodes = append(t.nodes, *n)
		idx = uint(len(t.nodes) - 1)
	}
	if n.level != 0 {
		(&t.nodes[n.left]).parent = idx
		(&t.nodes[n.right]).parent = idx
	}
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
	n := src.nodes[idx]
	if n.left > 0 {
		n.left = t.takeOwnership(src, n.left)
	}
	if n.right > 0 {
		n.right = t.takeOwnership(src, n.right)
	}
	return t.cp(&n)
}

func (t *Tree) free(src *Tree, idx uint, recursive bool) {
	if src != t {
		// Don't free nodes for a different tree
		return
	}
	if idx == 0 {
		return
	}
	n := &src.nodes[idx]
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
	if (op == common) || (op == or && bul) || (op == and && !bul) {
		t.free(at, a, true)
		idx = 0
		return
	}
	idx = a
	return
}

func (t *Tree) collision(at, bt *Tree, a, b uint, aul, bul bool, op operation) uint {
	an, bn := &at.nodes[a], &bt.nodes[b]
	var below, includes, above, boundBelow, boundAbove, unbounded bool
	switch op {
	case common:
		if aul != bul || an.incl != bn.incl || an.ul != bn.ul {
			t.free(at, a, true)
			t.free(bt, b, true)
			return 0
		}
		if !aul {
			// Check subsequent node to see if we should keep this one
			as := at.nextLeaf(a)
			bs := bt.nextLeaf(b)
			if !(&at.nodes[as]).Equals(&bt.nodes[bs]) {
				return 0
			}
		} else {
			ap := at.previousLeaf(a)
			bp := bt.previousLeaf(b)
			if !(&at.nodes[ap]).Equals(&bt.nodes[bp]) {
				return 0
			}
		}
		op = and
		fallthrough
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
	an, bn := &at.nodes[a], &bt.nodes[b]
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
		left, right = t.takeOwnership(lt, left), t.takeOwnership(rt, right)
		idx = t.node(prefix, level, left, right, (&t.nodes[left]).ul != (&t.nodes[right]).ul, false)
	}
	return
}

func (t *Tree) merge(at, bt *Tree, a, b uint, aul, bul bool, op operation) (idx uint) {
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
	an, bn := &at.nodes[a], &bt.nodes[b]
	switch {
	case an.level > bn.level:
		if !IsPrefixAt(bn.prefix, an.prefix, an.level) {
			// disjoint trees
			if op == common {
				t.free(at, a, true)
				t.free(bt, b, true)
				idx = 0
				return
			}
			// encompass with a node for common prefix
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		// b is somewhere under a
		var (
			left, right uint
			lt, rt      *Tree
		)
		// Won't be needing a again
		a_left, a_right, a_prefix, a_level := an.left, an.right, an.prefix, an.level
		t.free(at, a, false)
		if ZeroAt(bn.prefix, a_level) {
			rul := bul != bn.ul
			// b is under the left side of a
			left, lt = t.merge(at, bt, a_left, b, aul, bul, op), t
			right, rt = t.overlap(at, a_right, rul, op), at
		} else {
			// b is under the right side of a
			rul := aul != (&at.nodes[a_left]).ul
			left, lt = t.overlap(at, a_left, bul, op), at
			right, rt = t.merge(at, bt, a_right, b, rul, bul, op), t
		}
		if left == 0 {
			idx = t.takeOwnership(rt, right)
		} else if right == 0 {
			idx = t.takeOwnership(lt, left)
		} else {
			left, right = t.takeOwnership(lt, left), t.takeOwnership(rt, right)
			idx = t.node(a_prefix, a_level, left, right,
				(&t.nodes[left]).ul != (&t.nodes[right]).ul,
				false)
		}
		return
	case bn.level > an.level:
		if !IsPrefixAt(an.prefix, bn.prefix, bn.level) {
			// disjoint trees
			if op == common {
				t.free(at, a, true)
				t.free(bt, b, true)
				idx = 0
				return
			}
			// encompass with a node for common prefix
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		// a is somewhere under b
		var (
			left, right uint
			lt, rt      *Tree
		)
		b_left, b_right, b_prefix, b_level := bn.left, bn.right, bn.prefix, bn.level
		t.free(bt, b, false)
		if ZeroAt(an.prefix, b_level) {
			// a is under the left side of b
			lul := aul != an.ul
			left, lt = t.merge(at, bt, a, b_left, aul, bul, op), t
			right, rt = t.overlap(bt, b_right, lul, op), bt
		} else {
			rul := bul != (&bt.nodes[b_right]).ul
			left, lt = t.overlap(bt, b_left, aul, op), bt
			right, rt = t.merge(at, bt, a, b_right, aul, rul, op), t
		}
		if left == 0 {
			idx = t.takeOwnership(rt, right)
		} else if right == 0 {
			idx = t.takeOwnership(lt, left)
		} else {
			left, right = t.takeOwnership(lt, left), t.takeOwnership(rt, right)
			idx = t.node(b_prefix, b_level, left, right,
				(&t.nodes[left]).ul != (&t.nodes[right]).ul,
				false)
		}
		return
	default: // equal level
		a_left, a_right, b_left, b_right := an.left, an.right, bn.left, bn.right
		prefix, level := an.prefix, an.level
		if an.prefix != bn.prefix {
			// disjoint trees
			if op == common {
				t.free(at, a, true)
				t.free(bt, b, true)
				idx = 0
				return
			}
			// encompass with a node for common prefix
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		if an.level == 0 {
			// Two representations of the same leaf
			idx = t.collision(at, bt, a, b, aul, bul, op)
			return
		}
		// Two internal nodes with same prefix; merge left with left, right with right
		lul := aul != (&at.nodes[a_left]).ul
		rul := bul != (&bt.nodes[b_left]).ul
		left := t.merge(at, bt, a_left, b_left, aul, bul, op)
		right := t.merge(at, bt, a_right, b_right, lul, rul, op)
		// Merge takes ownership, so need to try again here
		if left == 0 {
			idx = right
			return
		}
		if right == 0 {
			idx = left
			return
		}
		newul := (&t.nodes[left]).ul != (&t.nodes[right]).ul
		idx = t.node(prefix, level, left, right, newul, false)
		return
	}
}

func (t *Tree) Clear() {
	t.root = 0
	t.nodes = t.nodes[:1]
	t.ul = false
	t.numfree = 0
	t.nextfree = 0
}

func (t *Tree) mergeRoot(at, bt *Tree, a, b uint, aul, bul bool, op operation) {
	t.root = t.merge(at, bt, a, b, aul, bul, op)
	switch op {
	case and, common:
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

func (t *Tree) leftmostLeaf(a uint, ul bool) (uint, bool) {
	n := t.nodes[a]
	for n.level != 0 {
		a = n.left
		n = t.nodes[a]
	}
	return a, ul
}

func (t *Tree) rightmostLeaf(a uint, ul bool) (uint, bool) {
	n := t.nodes[a]
	for n.level != 0 {
		a = n.right
		ul = ul != t.nodes[n.left].ul
		n = t.nodes[a]
	}
	return a, ul
}

func (t *Tree) previousLeaf(a uint) uint {
	n := &t.nodes[a]
	if n.level != 0 {
		panic("Tried to call previousLeaf on a non-leaf")
	}
	for a != t.root {
		p := &t.nodes[n.parent]
		switch a {
		case p.left:
			// We came up the left side. Keep going up until we can take a left
			a = n.parent
			n = &t.nodes[a]
		case p.right:
			idx, _ := t.rightmostLeaf(p.left, false) // ul is ignored here
			return idx
		}
	}
	// This tree doesn't have a previous leaf
	return 0
}

func (t *Tree) nextLeaf(a uint) uint {
	var n, p *node
	n = &t.nodes[a]
	if n.level != 0 {
		panic("Tried to call nextLeaf on a non-leaf")
	}
	for a != t.root {
		p = &t.nodes[n.parent]
		switch a {
		case p.right:
			// We came up the right side. Keep going up until we can take a right
			a = n.parent
			n = &t.nodes[a]
		case p.left:
			idx, _ := t.leftmostLeaf(p.right, false) // ul is ignored here
			return idx
		}
	}
	// This tree doesn't have a previous leaf
	return 0
}

func (t *Tree) leftEdge(key uint64) (uint, bool) {
	var (
		lidx uint
		lul  bool
		idx  uint = t.root
		n    node = t.nodes[idx]
		ul   bool = t.ul
	)
	for n.level != 0 && IsPrefixAt(key, n.prefix, n.level) {
		switch {
		case ZeroAt(key, n.level):
			idx = n.left
		default:
			idx = n.right
			lidx = n.left
			lul = ul
			ul = ul != t.nodes[n.left].ul
		}
		n = t.nodes[idx]
	}
	switch {
	case idx == 0:
		return 0, t.ul
	case (n.level == 0 && n.boundAbove()), n.prefix > key:
		return t.rightmostLeaf(lidx, lul)
	default:
		return t.rightmostLeaf(idx, ul)
	}
}

func (t *Tree) rightEdge(key uint64) (uint, bool) {
	var (
		ridx uint
		rul  bool
		idx  uint = t.root
		n    node = t.nodes[idx]
		ul   bool = t.ul
	)
	for n.level != 0 && IsPrefixAt(key, n.prefix, n.level) {
		switch {
		case !ZeroAt(key, n.level):
			idx = n.right
			ul = ul != t.nodes[n.left].ul
		default:
			idx = n.left
			ridx = n.right
			rul = ul != t.nodes[n.left].ul
		}
		n = t.nodes[idx]
	}
	switch {
	case idx == 0:
		return 0, ul
	case (n.level == 0 && n.boundBelow()), n.prefix < key:
		return t.leftmostLeaf(ridx, rul)
	default:
		return t.leftmostLeaf(idx, ul)

	}
}
