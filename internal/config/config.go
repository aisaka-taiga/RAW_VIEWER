package config

type Config struct {
	LibraryRoot  string
	CacheDir     string
	ExifToolPath string
	SQLitePath   string
	ListenAddr   string
}

func Default() Config {
	return Config{
		LibraryRoot:  `C:\Photos`,
		CacheDir:     "data/thumbs",
		ExifToolPath: "exiftool", // Assume it's in the PATH
		SQLitePath:   "data/app.db",
		ListenAddr:   "127.0.0.1:50051",
	}
}
