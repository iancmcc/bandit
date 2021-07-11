package bandit

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"strings"

	"github.com/golang/snappy"
	"github.com/xlab/treeprint"
)

type (
	operation uint8
	node      struct {
		prefix uint64
		level  uint
		parent uint
		left   uint
		right  uint
		count  uint
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
	return fmt.Sprintf(`[ [%d] %b (%d) P: %d L:%d R:%d UL:%t INCL:%t ]`,
		n.level, n.prefix, n.prefix, n.parent, n.left, n.right, n.ul, n.incl)
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
	idx = t.cp(node{
		prefix: prefix,
		level:  level,
		left:   left,
		right:  right,
		ul:     ul,
		incl:   incl,
	})
	if idx == left || idx == right {
		panic("Cycle detected!")
	}
	return
}

func (t *Tree) cp(n node) (idx uint) {
	nn := n
	if nn.level == 0 {
		nn.count = 1
	}
	if t.numfree > 0 && t.nextfree == 0 {
		panic("tree free list leak")
	}
	if t.numfree > 0 {
		idx = t.nextfree
		if nn.left == idx || nn.right == idx {
			t.PrintTree()
			panic(fmt.Sprintf("Refusing to create a loop with idx %d", idx))
		}
		t.nextfree = (&t.nodes[idx]).left
		t.nodes[idx] = nn
		if t.nextfree == 0 {
			t.numfree = 1
		}
		t.numfree -= 1
	} else {
		/*
			if cap(t.nodes) == 12 {
				fmt.Println("Trying to append to an interval")
			}
		*/
		t.nodes = append(t.nodes, nn)
		idx = uint(len(t.nodes) - 1)
	}
	if n.level != 0 {
		l, r := &t.nodes[n.left], &t.nodes[n.right]
		l.parent, r.parent = idx, idx
		(&t.nodes[idx]).count = l.count + r.count
	}
	return
}

func (t *Tree) capEstimate() uint {
	return uint(len(t.nodes)) - t.numfree
}

func (t *Tree) ensureCapacity(n uint) {
	c := uint(cap(t.nodes))
	if n > c {
		out := make([]node, 0, n+c)
		copy(out, t.nodes)
		t.nodes = out[:len(t.nodes)]
	}
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
	if !recursive {
		t.nodes[idx], t.nextfree = node{left: t.nextfree}, idx
		t.numfree += 1
		return
	}
	n := &src.nodes[idx]
	stack := append(make([]uint, 0, n.count+2), idx)
	seen := make(map[uint]struct{})
	for len(stack) > 0 {
		idx, stack = stack[len(stack)-1], stack[:len(stack)-1]
		if _, ok := seen[idx]; !ok {
			n = &src.nodes[idx]
			if n.right != 0 {
				stack = append(stack, n.right)
			}
			if n.left != 0 {
				stack = append(stack, n.left)
			}
			seen[idx] = struct{}{}
			t.nodes[idx], t.nextfree = node{left: t.nextfree}, idx
			t.numfree += 1
		}
	}
}

func (t *Tree) overlap(at *Tree, a uint, bul bool, op operation) (idx uint) {
	if (op == or && bul) || (op == and && !bul) {
		t.free(at, a, true)
		idx = 0
		return
	}
	idx = t.takeOwnership(at, a)
	return
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
		t.free(at, a, true)
		t.free(bt, b, true)
		return 0
	case boundBelow == an.incl && unbounded == an.ul:
		if a == b && at == bt {
			return t.takeOwnership(at, a)
		}
		t.free(bt, b, true)
		return t.takeOwnership(at, a)
	case boundBelow == bn.incl && unbounded == bn.ul:
		if a == b && at == bt {
			return t.takeOwnership(bt, b)
		}
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
	)
	if ZeroAt(an.prefix, level) {
		lul := aul != an.ul
		left = t.overlap(at, a, bul, op)
		right = t.overlap(bt, b, lul, op)
	} else {
		rul := bul != bn.ul
		left = t.overlap(bt, b, aul, op)
		right = t.overlap(at, a, rul, op)
	}
	if left == 0 {
		idx = right
	} else if right == 0 {
		idx = left
	} else {
		idx = t.node(prefix, level, left, right, (&t.nodes[left]).ul != (&t.nodes[right]).ul, false)
	}
	return
}

func (t *Tree) PrintTree() {
	fmt.Println(t.String())
	fmt.Println("\nUL:", t.ul)
	tree := treeprint.New()
	t.addToTree(t.root, tree)
	fmt.Println(tree.String())
}

func (t *Tree) addToTree(a uint, tr treeprint.Tree) {
	n := &t.nodes[a]
	if n.level == 0 {
		tr.AddMetaNode(fmt.Sprintf("%d/%d", n.parent, a), fmt.Sprintf("%d (U:%t, I:%t)", n.prefix, n.ul, n.incl))
		return
	}
	tr = tr.AddMetaBranch(fmt.Sprintf("%d/%d", n.parent, a), fmt.Sprintf("%d (U:%t)", n.prefix, n.ul))
	t.addToTree(n.left, tr)
	t.addToTree(n.right, tr)
}

func (t *Tree) merge(at, bt *Tree, a, b uint, aul, bul bool, op operation) (idx uint) {
	/*
		log := func(s ...interface{}) {
			if op == or {
				fmt.Println(s...)
			}
		}
	*/
	if a == 0 && b == 0 {
		idx = 0
		return
	}
	if a == 0 {
		idx = t.overlap(bt, b, aul, op)
		return
	}
	if b == 0 {
		idx = t.overlap(at, a, bul, op)
		return
	}
	an, bn := &at.nodes[a], &bt.nodes[b]
	switch {
	case an.level > bn.level:
		if !IsPrefixAt(bn.prefix, an.prefix, an.level) {
			// disjoint trees
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		// b is somewhere under a
		var (
			left, right uint
		)
		// Won't be needing a again
		a_left, a_right, a_prefix, a_level := an.left, an.right, an.prefix, an.level
		var tofree uint
		if a != b || at != bt {
			tofree = a
		}
		if ZeroAt(bn.prefix, a_level) {
			rul := bul != bn.ul
			// b is under the left side of a
			left = t.merge(at, bt, a_left, b, aul, bul, op)
			right = t.overlap(at, a_right, rul, op)
		} else {
			// b is under the right side of a
			rul := aul != (&at.nodes[a_left]).ul
			left = t.overlap(at, a_left, bul, op)
			right = t.merge(at, bt, a_right, b, rul, bul, op)
		}
		if left == 0 {
			idx = right
		} else if right == 0 {
			idx = left
		} else {
			idx = t.node(a_prefix, a_level, left, right,
				(&t.nodes[left]).ul != (&t.nodes[right]).ul,
				false)
		}
		t.free(at, tofree, false)
		return
	case bn.level > an.level:
		if !IsPrefixAt(an.prefix, bn.prefix, bn.level) {
			// disjoint trees
			idx = t.join(at, bt, a, b, aul, bul, op)
			return
		}
		// a is somewhere under b
		var (
			left, right uint
		)
		b_left, b_right, b_prefix, b_level := bn.left, bn.right, bn.prefix, bn.level
		var tofree uint
		if a != b || at != bt {
			tofree = b
		}
		if ZeroAt(an.prefix, b_level) {
			// a is under the left side of b
			lul := aul != an.ul
			left = t.merge(at, bt, a, b_left, aul, bul, op)
			right = t.overlap(bt, b_right, lul, op)
		} else {
			rul := bul != (&bt.nodes[b_right]).ul
			left = t.overlap(bt, b_left, aul, op)
			right = t.merge(at, bt, a, b_right, aul, rul, op)
		}
		if left == 0 {
			idx = right
		} else if right == 0 {
			idx = left
		} else {
			idx = t.node(b_prefix, b_level, left, right,
				(&t.nodes[left]).ul != (&t.nodes[right]).ul,
				false)
		}
		t.free(bt, tofree, false)
		return
	default: // equal level
		a_left, a_right, b_left, b_right := an.left, an.right, bn.left, bn.right
		prefix, level := an.prefix, an.level
		if an.prefix != bn.prefix {
			// disjoint trees
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
	if len(t.nodes) < 1 {
		t.nodes = append(t.nodes, node{})
	}
	t.nodes = t.nodes[:1]
	t.ul = false
	t.numfree = 0
	t.nextfree = 0
}

func (t *Tree) mergeRoot(at, bt *Tree, a, b uint, aul, bul bool, op operation) {
	an, bn := at.capEstimate(), bt.capEstimate()
	switch op {
	case and:
		if an > bn {
			t.ensureCapacity(an)
		} else {
			t.ensureCapacity(bn)
		}
	case or, xor:
		t.ensureCapacity(an + bn)
	}

	t.root = t.merge(at, bt, a, b, aul, bul, op)
	(&t.nodes[t.root]).parent = 0
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

func (t *Tree) leftmostLeaf(a uint, ul bool) (uint, bool) {
	if a == t.root && ul {
		// Unbounded left
		return 0, true
	}
	n := &t.nodes[a]
	for n.level != 0 {
		a = n.left
		n = &t.nodes[a]
	}
	return a, ul
}

func (t *Tree) rightmostLeaf(a uint, ul bool) (uint, bool) {
	var isroot = a == t.root
	n := &t.nodes[a]
	for n.level != 0 {
		a = n.right
		ul = ul != (&t.nodes[n.left]).ul
		n = &t.nodes[a]
	}
	if isroot && !ul {
		// Unbounded right
		return 0, true
	}
	return a, ul
}

func (t *Tree) ulAt(a uint) bool {
	n := &t.nodes[a]
	ul := t.ul
	for a != t.root {
		if n.parent == 0 {
			t.PrintTree()
			panic(fmt.Sprintf("INCORRECT PARENT: %v", n))
		}
		p := &t.nodes[n.parent]
		if p.right == a {
			ul = ul != (&t.nodes[p.left]).ul
		}
		a = n.parent
		n = p
	}
	return ul
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
			idx, _ := t.rightmostLeaf(p.left, t.ulAt(p.left)) // ul is ignored here
			return idx
		default:
			panic("Inconsistency: child has parent, parent doesn't have child")
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
			idx, _ := t.leftmostLeaf(p.right, t.ulAt(p.right)) // ul is ignored here
			return idx
		default:
			panic("Inconsistency: child has parent, parent doesn't have child")
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

func (n *node) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	err := enc.Encode(n.prefix)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.level)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.parent)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.left)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.right)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.count)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.ul)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(n.incl)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (n *node) GobDecode(buf []byte) error {
	w := bytes.NewBuffer(buf)
	enc := gob.NewDecoder(w)
	err := enc.Decode(&n.prefix)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.level)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.parent)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.left)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.right)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.count)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.ul)
	if err != nil {
		return err
	}
	err = enc.Decode(&n.incl)
	if err != nil {
		return err
	}
	return nil
}

func (t *Tree) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	err := enc.Encode(t.root)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(t.nextfree)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(t.numfree)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(t.ul)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(t.nodes)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (t *Tree) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	enc := gob.NewDecoder(r)
	err := enc.Decode(&t.root)
	if err != nil {
		return err
	}
	err = enc.Decode(&t.nextfree)
	if err != nil {
		return err
	}
	err = enc.Decode(&t.numfree)
	if err != nil {
		return err
	}
	err = enc.Decode(&t.ul)
	if err != nil {
		return err
	}
	err = enc.Decode(&t.nodes)
	if err != nil {
		return err
	}
	return nil
}

func (t *Tree) Dump(w io.Writer) error {
	bw := snappy.NewBufferedWriter(w)
	defer bw.Close()
	enc := gob.NewEncoder(bw)
	return enc.Encode(t)
}

func LoadTree(r io.Reader) (*Tree, error) {
	var t Tree
	enc := gob.NewDecoder(snappy.NewReader(r))
	if err := enc.Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}
