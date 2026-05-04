package domain

// AppConfig groups the program's configuration settings to facilitate parameter
// passing
type AppConfig struct {
	Format    string
	Rate      int
	InputFile string
	URLs      []string
}
