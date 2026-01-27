package credential

type Credential struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ProcessingStats struct {
	TotalLines       int
	ValidCredentials int
	DuplicatesFound  int
	LinesIgnored     int
}

type ProcessingOptions struct {
	EnableDeduplication bool
	SaveDuplicates      bool
	DuplicatesFile      string
	Quiet               bool
	BatchSize           int
}

type ProcessingResult struct {
	Credentials []Credential
	Stats       ProcessingStats
	Duplicates  []string
}

type URLNormalizer interface {
	Normalize(rawURL string) string
}

type CredentialProcessor interface {
	ProcessLine(line string) (*Credential, error)
	ProcessFile(filename string, opts ProcessingOptions) (*ProcessingResult, error)
	ProcessDirectory(dirname string, opts ProcessingOptions) (map[string]*ProcessingResult, error)
	ProcessFileStreaming(filename string, opts ProcessingOptions, batchWriter BatchWriter) (*ProcessingStats, error)
}

type BatchWriter interface {
	WriteBatch(credentials []Credential) error
	Flush() error
}
