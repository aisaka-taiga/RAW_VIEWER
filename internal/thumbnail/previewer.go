package thumbnail

import "context"

type Previewer struct {
	ExifToolPath string
}

func (p Previewer) GetPreviewBytes(ctx context.Context, path string, size int) ([]byte, error) {
	return ExifToolPreviewer{ExePath: p.ExifToolPath}.GetPreviewBytes(ctx, path, size)
}
