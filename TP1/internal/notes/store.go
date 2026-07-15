package notes

type NoteStore interface {
	Add(Note) error
	List() ([]Note, error)
}
