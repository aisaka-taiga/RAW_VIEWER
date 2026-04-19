package metadata

type Reader struct {
	ExifToolPath string
}

func (r Reader) Read(path string) (map[string]string, string, error) {
	return ExifToolReader{ExePath: r.ExifToolPath}.Read(path)
}
