package internal

import (
	"encoding/gob"
	"fmt"
	"os"
	"sync"
	"time"
)

type Persister struct {
	filename string
	mu       sync.Mutex
}

func NewPersister(filename string) *Persister {
	return &Persister{filename: filename}
}

// Save writes the map to a binary file
func (p *Persister) Save(data map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Create(p.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	if err := enc.Encode(data); err != nil {
		return err
	}
	fmt.Println("💾 Snapshot saved to", p.filename)
	return nil
}

// Load reads the map back into memory
func (p *Persister) Load() map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()

	data := make(map[string]string)

	file, err := os.Open(p.filename)
	if err != nil {
		fmt.Println("No snapshot found, starting fresh")
		return data
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		fmt.Println("❌ Error decoding snapshot:", err)
	} else {
		fmt.Println("✅ Snapshot loaded from", p.filename)
	}

	return data
}

// AutoSave periodically writes the data
func (p *Persister) AutoSave(getData func() map[string]string) {
	for {
		time.Sleep(5 * time.Second)
		data := getData()
		if err := p.Save(data); err != nil {
			fmt.Println("Error during autosave:", err)
		}
	}
}
