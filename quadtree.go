package quadtree

import (
	"fmt"
)

const (
	DEFAULT_MAX_OBJECTS = 128
	DEFAULT_MAX_LEVELS  = 5
)

/*
 *  rectangle bounds
 *     (0,0)
 *     +--------------------------->X
 *     |  (x, y)
 *     |    +-----------+
 *     |    |           | height
 *     |    +-----------+
 *     |         width
 *     |
 *     |
 *     V Y
 *
 */
type Rectangle struct {
	X      int32 // left top
	Y      int32 // left top
	Width  int32
	Height int32
}

func MakeRect(x int32, y int32, width int32, height int32) Rectangle {
	return Rectangle{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

func (rect Rectangle) Contain(other Rectangle) bool {
	if other.X >= rect.X &&
		other.Y >= rect.Y &&
		other.X+other.Width <= rect.X+rect.Width &&
		other.Y+other.Height <= rect.Y+rect.Height {
		return true
	}
	return false
}

// 正方形内切圆
func (rect Rectangle) ToCircle() (int32, int32, int32) {
	if rect.Width != rect.Height {
		return -1, -1, -1
	}
	r := rect.Width / 2
	return rect.X + r, rect.Y + r, r
}

func CircleBounds(cx int32, cy int32, radius int32) Rectangle {
	return Rectangle{
		X:      cx - radius,
		Y:      cy - radius,
		Width:  radius * 2,
		Height: radius * 2,
	}
}

type Object struct {
	Id     int64
	Bounds Rectangle
	Data   interface{}
}

func NewObject(id int64, bounds Rectangle, data interface{}) Object {
	return Object{
		Id:     id,
		Bounds: bounds,
		Data:   data,
	}
}

type quadNode struct {
	objects    map[int64]Object
	children   []*quadNode
	bounds     Rectangle
	maxLevels  int
	maxObjects int
	level      int
}

func newQuadNode(x int32, y int32, width int32, height int32, level int, maxObjects int, maxLevels int) *quadNode {
	return &quadNode{
		bounds:     Rectangle{x, y, width, height},
		objects:    make(map[int64]Object),
		children:   nil,
		maxLevels:  maxLevels,
		maxObjects: maxObjects,
		level:      level,
	}
}

type QuadTree struct {
	maxObjects int
	maxLevels  int
	root       *quadNode
}

func NewQuadTree(width int32, height int32, maxObjects int, maxLevels int) *QuadTree {
	if maxObjects <= 0 {
		maxObjects = DEFAULT_MAX_OBJECTS
	}
	if maxLevels <= 0 {
		maxLevels = DEFAULT_MAX_LEVELS
	}
	return &QuadTree{
		maxObjects: maxObjects,
		maxLevels:  maxLevels,
		root:       newQuadNode(0, 0, width, height, 1, maxObjects, maxLevels),
	}
}

func (t *QuadTree) Insert(obj Object) {
	t.root.insert(obj)
}

func (t *QuadTree) Retrieve(bound Rectangle, fn func(obj Object)) {
	t.root.retrieve(bound, fn)
}

func (t *QuadTree) Foreach(fn func(obj Object)) {
	t.root.foreach(fn)
}

func (t *QuadTree) Check(bound Rectangle, fn func(obj Object) bool) bool {
	return t.root.check(bound, fn)
}

func (t *QuadTree) Remove(bounds Rectangle, id int64) {
	t.root.remove(bounds, id)
}

func (t *QuadTree) Show() {
	levels := make([][]*quadNode, t.maxLevels+1)
	t.root.show(levels)
	for _, nodes := range levels {
		for _, node := range nodes {
			fmt.Printf("level: %d, bound: %v, objects: %d\n", node.level, node.bounds, len(node.objects))
		}
	}
}

func (node *quadNode) show(levels [][]*quadNode) {
	for _, child := range node.children {
		child.show(levels)
	}
	levels[node.level] = append(levels[node.level], node)
}

func (node *quadNode) split() {
	nextLevel := node.level + 1
	subWidth := node.bounds.Width / 2
	subHeight := node.bounds.Height / 2
	x := node.bounds.X
	y := node.bounds.Y
	node.children = make([]*quadNode, 4)

	// top right node
	node.children[0] = newQuadNode(x+subWidth, y, subWidth, subHeight, nextLevel, node.maxObjects, node.maxLevels)

	// top left node
	node.children[1] = newQuadNode(x, y, subWidth, subHeight, nextLevel, node.maxObjects, node.maxLevels)

	// bottom left node
	node.children[2] = newQuadNode(x, y+subHeight, subWidth, subHeight, nextLevel, node.maxObjects, node.maxLevels)

	// bottom right node
	node.children[3] = newQuadNode(x+subWidth, y+subHeight, subWidth, subHeight, nextLevel, node.maxObjects, node.maxLevels)

}

// getIndex - Determine which quadrant the object belongs to (0-3)
func (node *quadNode) getIndex(rect Rectangle) int {

	// index of the subnode (0-3), or -1 if pRect cannot completely fit within a subnode and is part of the parent node
	index := -1

	midX := node.bounds.X + (node.bounds.Width / 2)
	midY := node.bounds.Y + (node.bounds.Height / 2)

	//rect can completely fit within the top quadrants
	topQuadrant := (rect.Y < midY) && (rect.Y+rect.Height < midY)

	//rect can completely fit within the bottom quadrants
	bottomQuadrant := (rect.Y > midY)

	//rect can completely fit within the left quadrants
	if (rect.X < midX) && (rect.X+rect.Width < midX) {
		if topQuadrant {
			index = 1
		} else if bottomQuadrant {
			index = 2
		}

	} else if rect.X > midX {
		//rect can completely fit within the right quadrants
		if topQuadrant {
			index = 0
		} else if bottomQuadrant {
			index = 3
		}
	}
	return index
}

func (node *quadNode) insert(obj Object) {
	if len(node.children) > 0 {
		if index := node.getIndex(obj.Bounds); index != -1 {
			node.children[index].insert(obj)
			return
		}
	}

	node.objects[obj.Id] = obj
	if len(node.objects) > node.maxObjects && node.level < node.maxLevels {
		if len(node.children) == 0 {
			node.split()
		}
		for _, obj := range node.objects {
			if index := node.getIndex(obj.Bounds); index != -1 {
				node.children[index].insert(obj)
				delete(node.objects, obj.Id)
			}
		}
	}
}

func (node *quadNode) foreach(fn func(obj Object)) {
	for _, child := range node.children {
		child.foreach(fn)
	}
	for _, obj := range node.objects {
		fn(obj)
	}
}

func (node *quadNode) retrieve(bounds Rectangle, fn func(obj Object)) {
	if len(node.children) > 0 {
		if index := node.getIndex(bounds); index != -1 {
			node.children[index].retrieve(bounds, fn)
		} else {
			for _, child := range node.children {
				child.retrieve(bounds, fn)
			}
		}
	}
	for _, obj := range node.objects {
		fn(obj)
	}
}

func (node *quadNode) check(bounds Rectangle, fn func(obj Object) bool) bool {
	for _, obj := range node.objects {
		if fn(obj) {
			return true
		}
	}
	if len(node.children) > 0 {
		if index := node.getIndex(bounds); index != -1 {
			return node.children[index].check(bounds, fn)
		} else {
			for _, child := range node.children {
				if child.check(bounds, fn) {
					return true
				}
			}
		}
	}
	return false
}

func (node *quadNode) remove(bounds Rectangle, id int64) {
	if _, present := node.objects[id]; present {
		delete(node.objects, id)
		return
	}
	if len(node.children) > 0 {
		if idx := node.getIndex(bounds); idx != -1 {
			node.children[idx].remove(bounds, id)
			return
		}
	}
}
