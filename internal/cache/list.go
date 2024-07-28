package cache

type List interface {
	Len() int
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value interface{}
	Next  *ListItem
	Prev  *ListItem
}

type list struct {
	FirstNode *ListItem
	LastNode  *ListItem
	Size      int
}

func NewList() List {
	return new(list)
}

func (l *list) Len() int {
	return l.Size
}

func (l *list) Front() *ListItem {
	return l.FirstNode
}

func (l *list) Back() *ListItem {
	return l.LastNode
}

func (l *list) PushFront(v interface{}) *ListItem {
	newItem := &ListItem{
		Value: v,
		Next:  nil,
		Prev:  nil,
	}

	l.Size++

	if l.FirstNode == nil {
		l.FirstNode = newItem
		l.LastNode = newItem
		return newItem
	}

	exFirstNode := l.FirstNode
	newItem.Next = exFirstNode
	l.FirstNode = newItem
	exFirstNode.Prev = newItem

	return newItem
}

func (l *list) PushBack(v interface{}) *ListItem {
	if l.LastNode == nil {
		return l.PushFront(v)
	}

	newItem := &ListItem{
		Value: v,
		Next:  nil,
		Prev:  nil,
	}

	exLastNode := l.LastNode
	newItem.Prev = exLastNode
	l.LastNode = newItem
	exLastNode.Next = newItem
	l.Size++
	return newItem
}

func (l *list) Remove(i *ListItem) {
	if l.Size == 0 {
		return
	}

	if i == nil {
		return
	}

	prevItem := i.Prev
	nextItem := i.Next

	if prevItem == nil {
		l.FirstNode = nextItem
	} else {
		prevItem.Next = nextItem
	}

	if nextItem == nil {
		l.LastNode = prevItem
	} else {
		nextItem.Prev = prevItem
	}

	l.Size--
}

func (l *list) MoveToFront(i *ListItem) {
	if l.Size <= 1 {
		return
	}

	if i != l.FirstNode {
		l.PushFront(i.Value)
		l.Remove(i)
	}
}
