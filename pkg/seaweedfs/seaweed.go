package seaweedfs

import "fmt"

type Seaweed struct {
	cfg    *Config
	master *Master
	filers []*Filer
}

func NewSeaweed(cfg *Config) (*Seaweed, error) {
	s := &Seaweed{
		cfg: cfg,
	}

	master, err := NewMaster(cfg.MasterURL)
	if err != nil {
		return nil, fmt.Errorf("new master: %w", err)
	}

	master.SetDebug(cfg.debug)

	s.master = master

	for _, filerURL := range cfg.FilerURLs {
		filer, err := NewFiler(filerURL)
		if err != nil {
			return nil, fmt.Errorf("new filer: %w", err)
		}

		filer.SetDebug(cfg.debug)

		s.filers = append(s.filers, filer)
	}

	return s, nil
}

func (s *Seaweed) Master() *Master {
	return s.master
}

func (s *Seaweed) Filers() []*Filer {
	return s.filers
}
