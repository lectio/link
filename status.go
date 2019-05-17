package link

// TraversalStatus determines the state of a link traversal
type TraversalStatus interface {
	Attempted() bool
	Traversable() bool
	Link() Link
	Error() error
}

type traversalState struct {
	attempted   bool
	traversable bool
	link        Link
	err         error
}

func (lts *traversalState) Attempted() bool {
	return lts.attempted
}

func (lts *traversalState) Traversable() bool {
	return lts.traversable
}

func (lts *traversalState) Link() Link {
	return lts.link
}

func (lts *traversalState) Error() error {
	return lts.err
}
