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
}
