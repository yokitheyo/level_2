package wget

type DownloadJob struct {
	URL      string
	Depth    int
	FilePath string
}

type Resource struct {
	URL  string
	Data []byte
}
