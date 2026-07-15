package notes

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type JSONLStore struct {
	Path string
}

func NewJSONLStore() (*JSONLStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".mira")

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	return &JSONLStore{
		Path: filepath.Join(dir, "notes.jsonl"),
	}, nil
}

func (s *JSONLStore) Add(note Note) error {

	file, err := os.OpenFile(s.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(note)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(data) + "\n")
	return err
}

func (s *JSONLStore) List() ([]Note, error) {

	file, err := os.Open(s.Path)

	if os.IsNotExist(err) {
		return []Note{}, nil
	}

	if err != nil {
		return nil, err
	}

	defer file.Close()

	var notes []Note

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		var note Note

		if err := json.Unmarshal(scanner.Bytes(), &note); err == nil {
			notes = append(notes, note)
		}
	}

	return notes, scanner.Err()
}
