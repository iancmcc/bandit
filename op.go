package bandit

import (
	"strconv"
	"strings"
)

type operation uint8

const (
	and operation = iota
	or
	xor
)

const (
	empty    = "(Ø)"
	infinite = "(-∞, ∞)"
)

type (
	node struct {
		prefix uint64
		left   uint
		right  uint
		level  uint
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

func (t *Tree) node(prefix uint64, level, left, right uint, ul, incl bool) uint {
	return t.cp(node{
		prefix: prefix,
		level:  level,
		left:   left,
		right:  right,
		ul:     ul,
		incl:   incl,
	})
}

func (t *Tree) cp(n node) uint {
	if t.numfree > 0 {
		idx := t.nextfree
		t.nextfree = t.nodes[idx].left
		t.nodes[idx] = n
		t.numfree -= 1
		return idx
	}
	t.nodes = append(t.nodes, n)
	return uint(len(t.nodes) - 1)
}

func (t *Tree) takeOwnership(src *Tree, idx uint) uint {
	if src == t {
		// We're the owner already
		return idx
	}
	n := src.nodes[idx]
	if n.left > 0 {
		n.left = t.takeOwnership(src, n.left)
	}
	if n.right > 0 {
		n.right = t.takeOwnership(src, n.left)
	}
	return t.cp(n)
}

func (t *Tree) free(src *Tree, idx uint) {
	if src != t {
		// Don't free nodes for a different tree
		return
	}
	n := &t.nodes[idx]
	if n.left > 0 {
		t.free(t, n.left)
	}
	if n.right > 0 {
		t.free(t, n.right)
	}
	t.nodes[idx] = node{left: t.nextfree}
	t.nextfree = idx
	t.numfree += 1
}

func (t *Tree) overlap(at *Tree, a uint, bul bool, op operation) uint {
	if (op == or && bul) || (op == and && !bul) {
		t.free(at, a)
		return 0
	}
	return a
}

func (t *Tree) collision(at, bt *Tree, a, b uint, aul, bul bool, op operation) uint {
	an, bn := &at.nodes[a], &bt.nodes[b]
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
		t.free(at, a)
		t.free(bt, b)
		return 0
	case boundBelow == an.incl && unbounded == an.ul:
		t.free(bt, b)
		return t.takeOwnership(at, a)
	case boundBelow == bn.incl && unbounded == bn.ul:
		t.free(at, a)
		return t.takeOwnership(bt, b)
	}
	return t.node(an.prefix, 0, 0, 0, unbounded, boundBelow)
}

func (t *Tree) join(at, bt *Tree, a, b uint, aul, bul bool, op operation) uint {
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
		return t.takeOwnership(rt, right)
	}
	if right == 0 {
		return t.takeOwnership(lt, left)
	}
	return t.node(prefix, level, t.takeOwnership(lt, left), t.takeOwnership(rt, right), (&t.nodes[left]).ul != (&t.nodes[right]).ul, false)
}

func (t *Tree) merge(at, bt *Tree, a, b uint, aul, bul bool, op operation) uint {
	if a == 0 && b == 0 {
		return 0
	}
	if a == 0 {
		return t.takeOwnership(bt, t.overlap(bt, b, aul, op))
	}
	if b == 0 {
		return t.takeOwnership(at, t.overlap(at, a, bul, op))
	}
	an, bn := &at.nodes[a], &bt.nodes[b]
	switch {
	case an.level > bn.level:
		if !IsPrefixAt(bn.prefix, an.prefix, an.level) {
			// disjoint trees; encompass
			return t.join(at, bt, a, b, aul, bul, op)
		}
		// b is somewhere under a
		var (
			left, right uint
			lt, rt      *Tree
		)
		if ZeroAt(bn.prefix, an.level) {
			// b is under the left side of a
			lul := bul != (&t.nodes[an.right]).ul
			left, lt = t.merge(at, bt, an.left, b, aul, bul, op), t
			right, rt = t.overlap(at, an.right, lul, op), at
		} else {
			// b is under the right side of a
			rul := aul != (&t.nodes[an.left]).ul
			left, lt = t.overlap(at, an.left, bul, op), at
			right, rt = t.merge(at, bt, an.right, b, rul, bul, op), t
		}
		if left == 0 {
			return t.takeOwnership(rt, right)
		}
		if right == 0 {
			return t.takeOwnership(lt, left)
		}
		return t.node(an.prefix, an.level, t.takeOwnership(lt, left), t.takeOwnership(rt, right), (&t.nodes[left]).ul != (&t.nodes[right]).ul, false)
	case bn.level > an.level:
		if !IsPrefixAt(an.prefix, bn.prefix, bn.level) {
			// disjoint trees; encompass
			return t.join(at, bt, a, b, aul, bul, op)
		}
		// a is somewhere under b
		var (
			left, right uint
			lt, rt      *Tree
		)
		if ZeroAt(an.prefix, bn.level) {
			// a is under the left side of b
			lul := aul != (&t.nodes[an.left]).ul
			left, lt = t.merge(at, bt, a, bn.left, aul, bul, op), t
			right, rt = t.overlap(bt, bn.right, lul, op), bt
		} else {
			// a is under the right side of b
			rul := bul != (&t.nodes[an.right]).ul
			left, lt = t.overlap(bt, bn.left, aul, op), bt
			right, rt = t.merge(at, bt, a, bn.right, aul, rul, op), t
		}
		if left == 0 {
			return t.takeOwnership(rt, right)
		}
		if right == 0 {
			return t.takeOwnership(lt, left)
		}
		return t.node(bn.prefix, bn.level, t.takeOwnership(lt, left), t.takeOwnership(rt, right), (&t.nodes[left]).ul != (&t.nodes[right]).ul, false)
	default: // equal level
		if an.prefix != bn.prefix {
			// disjoint trees; encompass
			return t.join(at, bt, a, b, aul, bul, op)
		}
		if an.level == 0 {
			// Two representations of the same leaf
			return t.collision(at, bt, a, b, aul, bul, op)
		}
		// Two internal nodes with same prefix; merge left with left, right with right
		lul := aul != (&t.nodes[an.left]).ul
		rul := bul != (&t.nodes[an.right]).ul
		left := t.merge(at, bt, an.left, bn.left, aul, bul, op)
		right := t.merge(at, bt, an.right, bn.right, lul, rul, op)
		// Merge takes ownership, so need to try again here
		if left == 0 {
			return right
		}
		if right == 0 {
			return left
		}
		// At this point, the two original nodes just need to be merged into
		// one, potentially with a different left and right. Find the one
		// that's this tree and update it.
		newprefix := an.prefix
		newlevel := an.level
		newul := (&t.nodes[left]).ul != (&t.nodes[right]).ul
		var merged uint
		switch t {
		case at:
			merged = a
		case bt:
			merged = b
		default:
			return t.node(newprefix, newlevel, left, right, newul, false)
		}
		t.nodes[merged] = node{
			left:   left,
			right:  right,
			prefix: newprefix,
			level:  newlevel,
			ul:     newul,
			incl:   false,
		}
		return merged
	}
}

func (t *Tree) Reset() {
	t.root = 0
	t.nodes = t.nodes[:1]
	t.ul = false
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
	if t.root == 0 {
		if t.ul {
			return infinite
		}
		return empty
	}
	var (
		sb      strings.Builder
		stack   = make([]uint, 0, len(t.nodes)-int(t.numfree))
		ulstack = make([]bool, 0, len(t.nodes)-int(t.numfree))
		n       uint
		cur     node
		ul      bool
	)
	stack = append(stack, t.root)
	ulstack = append(ulstack, t.ul)
	for l := 1; l > 0; l = len(stack) {
		n, stack = stack[l-1], stack[:l-1]
		ul, ulstack = ulstack[l-1], ulstack[:l-1]
		cur = t.nodes[n]
		if cur.level != 0 {
			// This is a branch, so add its children (right first, so we visit
			// left first)
			stack = append(stack, cur.right, cur.left)
			// Reuse cur for left
			cur = t.nodes[cur.left]
			ulstack = append(ulstack, ul != cur.ul, ul)
			continue
		}
		// cur is a leaf
		v := strconv.FormatUint(cur.prefix, 10)
		switch {
		case cur.incl && cur.ul:
			// bound below
			if ul {
				// Closing an interval
				sb.WriteString(v)
				sb.WriteString(")")
			} else {
				// Opening an interval
				sb.WriteString("[")
				sb.WriteString(v)
				sb.WriteString(", ")
			}
		case !cur.incl && cur.ul:
			// bound above
			if ul {
				// Closing an interval
				sb.WriteString(v)
				sb.WriteString("]")
			} else {
				// Opening an interval
				sb.WriteString("(")
				sb.WriteString(v)
				sb.WriteString(", ")
			}
		case cur.incl && !cur.ul:
			// bound both
			if ul {
				// Hole
				sb.WriteString(v)
				sb.WriteString("), ")
				sb.WriteString("(")
				sb.WriteString(v)
				sb.WriteString(", ")
			} else {
				// Point
				sb.WriteString("[")
				sb.WriteString(v)
				sb.WriteString(", ")
				sb.WriteString(v)
				sb.WriteString("]")
			}
		}
	}
	return sb.String()
}
