package seaweedfs

type Config struct {
	MasterURL string
	FilerURLs []string
	debug     bool
}

func NewConfig(masterURL string) *Config {
	return &Config{
		MasterURL: masterURL,
	}
}

func NewConfigWithFilerURL(masterURL string, filerURL string) *Config {
	return &Config{
		MasterURL: masterURL,
		FilerURLs: []string{filerURL},
	}
}

func (c *Config) Debug() *Config {
	c.debug = true

	return c
}

func (c *Config) AddFilerURL(filerURL string) *Config {
	c.FilerURLs = append(c.FilerURLs, filerURL)

	return c
}
