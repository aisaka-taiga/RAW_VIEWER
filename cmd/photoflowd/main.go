package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	pb "photo-viewer/gen"
	"photo-viewer/internal/config"
	"photo-viewer/internal/db"
	"photo-viewer/internal/indexer"
	"photo-viewer/internal/library"
	"photo-viewer/internal/metadata"
	"photo-viewer/internal/thumbnail"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedPhotoEngineServer
	store       *db.Store
	indexer     indexer.Indexer
	thumbs      *thumbnail.Service
	libraryRoot string
	prewarmChan chan string
}

func (s *server) warmRawCache(ctx context.Context, path string) {
	if s.thumbs == nil || !thumbnail.IsRaw(path) {
		return
	}
	if err := s.thumbs.WarmRaw(ctx, path, 384, 1024); err != nil {
		log.Printf("raw warm cache skipped for %s: %v", path, err)
	}
}

func (s *server) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true, Version: "0.1.0"}, nil
}

func (s *server) ScanFolder(ctx context.Context, req *pb.ScanFolderRequest) (*pb.ScanFolderResponse, error) {
	root := s.libraryRoot
	if req.GetFolderPath() != "" {
		root = req.GetFolderPath()
	}
	var queued uint32
	if err := library.Walk(root, func(path string) error {
		if err := s.indexer.IndexFile(path); err != nil {
			return err
		}
		s.warmRawCache(ctx, path)
		select {
		case s.prewarmChan <- path:
		default:
		}
		queued++
		return nil
	}); err != nil {
		return nil, err
	}
	return &pb.ScanFolderResponse{Queued: queued}, nil
}

func (s *server) ScanFolderStream(req *pb.ScanFolderRequest, stream pb.PhotoEngine_ScanFolderStreamServer) error {
	root := s.libraryRoot
	if req.GetFolderPath() != "" {
		root = req.GetFolderPath()
	}
	var paths []string
	if err := library.Walk(root, func(path string) error {
		paths = append(paths, path)
		return nil
	}); err != nil {
		return err
	}
	total := uint32(len(paths))
	
	for i, path := range paths {
		if err := s.indexer.IndexFile(path); err != nil {
			return err
		}
		s.warmRawCache(stream.Context(), path)
		// Send to persistent background prewarmer queue
		select {
		case s.prewarmChan <- path:
		default:
			// If buffer is somehow full (100k items), just skip or wait briefly
		}
		photo, err := s.store.GetPhoto(path)
		if err != nil {
			return err
		}
		if err := stream.Send(&pb.ScanFolderProgress{
			Scanned: uint32(i + 1),
			Total:   total,
			Photo: &pb.PhotoItem{
				Id:         photo.ID,
				Path:       photo.Path,
				FileName:   photo.FileName,
				Format:     photo.Format,
				Width:      uint32(photo.Width),
				Height:     uint32(photo.Height),
				CapturedAt: photo.CapturedAt,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) ListPhotos(ctx context.Context, req *pb.ListPhotosRequest) (*pb.ListPhotosResponse, error) {
	photos, total, err := s.store.ListPhotos(req.GetFolderPath(), req.GetLimit(), req.GetOffset(), req.GetSortBy(), req.GetDescending())
	if err != nil {
		return nil, err
	}
	items := make([]*pb.PhotoItem, 0, len(photos))
	for _, p := range photos {
		items = append(items, &pb.PhotoItem{
			Id:         p.ID,
			Path:       p.Path,
			FileName:   p.FileName,
			Format:     p.Format,
			Width:      uint32(p.Width),
			Height:     uint32(p.Height),
			CapturedAt: p.CapturedAt,
		})
	}
	return &pb.ListPhotosResponse{Items: items, Total: total}, nil
}

var thumbSemaphore chan struct{}
var backgroundSemaphore chan struct{}

func init() {
	// Dynamically scale worker pool based on CPU cores
	numCpus := runtime.NumCPU()
	
	// UI/Foreground limit
	limit := numCpus * 2
	if limit < 8 { limit = 8 }
	thumbSemaphore = make(chan struct{}, limit)

	// Background/Pre-warm limit (slightly lower to prioritize UI)
	bgLimit := numCpus 
	if bgLimit < 4 { bgLimit = 4 }
	backgroundSemaphore = make(chan struct{}, bgLimit)
}

func (s *server) GetThumbnail(ctx context.Context, req *pb.GetThumbnailRequest) (*pb.GetThumbnailResponse, error) {
	// Acquire semaphore to prevent memory exhaustion during parallel processing
	select {
	case thumbSemaphore <- struct{}{}:
		defer func() { <-thumbSemaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	photo, err := s.store.GetPhoto(req.GetPhotoId())
	if err != nil {
		return nil, err
	}
	entry, data, err := s.thumbs.Get(ctx, photo.Path, int(req.GetSize()))
	if err != nil {
		return nil, err
	}
	return &pb.GetThumbnailResponse{
		Data:      data,
		MimeType:  entry.MimeType,
		FromCache: entry.FromDisk,
	}, nil
}

func (s *server) GetMetadata(ctx context.Context, req *pb.GetMetadataRequest) (*pb.GetMetadataResponse, error) {
	meta, err := s.store.GetMetadata(req.GetPhotoId())
	if err != nil && db.IsNotFound(err) {
		if err := s.indexer.IndexFile(req.GetPhotoId()); err != nil {
			return nil, err
		}
		meta, err = s.store.GetMetadata(req.GetPhotoId())
	}
	if err != nil {
		return nil, err
	}
	fields := []*pb.MetadataField{
		{Key: "Make", Value: meta.Make},
		{Key: "Model", Value: meta.Model},
		{Key: "ISO", Value: meta.ISO},
		{Key: "Aperture", Value: meta.Aperture},
		{Key: "ShutterSpeed", Value: meta.ShutterSpeed},
		{Key: "FocalLength", Value: meta.FocalLength},
		{Key: "LensModel", Value: meta.LensModel},
	}
	return &pb.GetMetadataResponse{PhotoId: req.GetPhotoId(), Fields: fields}, nil
}

func (s *server) DeletePhoto(ctx context.Context, req *pb.DeletePhotoRequest) (*pb.DeletePhotoResponse, error) {
	if err := s.store.DeletePhoto(req.GetPhotoId()); err != nil {
		return nil, err
	}
	return &pb.DeletePhotoResponse{Ok: true}, nil
}

func (s *server) MoveToRejected(ctx context.Context, req *pb.MoveToRejectedRequest) (*pb.MoveToRejectedResponse, error) {
	photo, err := s.store.GetPhoto(req.GetPhotoId())
	if err != nil {
		return nil, err
	}

	oldPath := photo.Path
	dir := filepath.Dir(oldPath)
	rejectDir := filepath.Join(dir, "_rejected")

	if err := os.MkdirAll(rejectDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create _rejected folder: %w", err)
	}

	newPath := filepath.Join(rejectDir, filepath.Base(oldPath))
	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("failed to move file: %w", err)
	}

	// Delete from database after moving the file
	if err := s.store.DeletePhoto(req.GetPhotoId()); err != nil {
		return nil, fmt.Errorf("failed to delete from db: %w", err)
	}

	return &pb.MoveToRejectedResponse{Ok: true, NewPath: newPath}, nil
}

func (s *server) PreloadNext(ctx context.Context, req *pb.PreloadNextRequest) (*pb.PreloadNextResponse, error) {
	return &pb.PreloadNextResponse{Queued: req.GetCount()}, nil
}

func main() {
	cfg := config.Default()
	flag.StringVar(&cfg.LibraryRoot, "library-root", cfg.LibraryRoot, "root folder containing photos")
	flag.StringVar(&cfg.CacheDir, "cache-dir", cfg.CacheDir, "thumbnail cache directory")
	flag.StringVar(&cfg.ExifToolPath, "exiftool-path", cfg.ExifToolPath, "path to exiftool executable")
	flag.StringVar(&cfg.SQLitePath, "sqlite-path", cfg.SQLitePath, "path to sqlite database file")
	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "gRPC listen address")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
		log.Fatal(err)
	}

	sqlDB, err := indexer.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		log.Fatal(err)
	}
	store := &db.Store{DB: sqlDB}
	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	s := &server{
		store: store,
		indexer: indexer.Indexer{
			Store: store,
			Meta:  metadata.Reader{ExifToolPath: cfg.ExifToolPath},
		},
		thumbs:      thumbnail.NewConfiguredService(cfg.CacheDir, cfg.ExifToolPath, store),
		libraryRoot: cfg.LibraryRoot,
		prewarmChan: make(chan string, 100000), // Massive buffer for background tasks
	}

	// Start persistent background pre-warm workers (Limited for better system stability)
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		go func() {
			for p := range s.prewarmChan {
				// Use backgroundSemaphore to avoid blocking the UI
				select {
				case backgroundSemaphore <- struct{}{}:
					if thumbnail.IsRaw(p) {
						if err := s.thumbs.WarmRaw(context.Background(), p, 384, 1024); err != nil {
							log.Printf("background raw warm failed for %s: %v", p, err)
						}
					} else {
						// During scan, only warm the grid thumbnail to keep CPU usage sane.
						s.thumbs.Get(context.Background(), p, 384)
					}
					<-backgroundSemaphore
				case <-time.After(1 * time.Second):
					// Skip if backend is too busy
				}
			}
		}()
	}

	lis, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(32<<20),
		grpc.MaxSendMsgSize(32<<20),
	)
	pb.RegisterPhotoEngineServer(grpcServer, s)

	go func() {
		<-time.After(50 * time.Millisecond)
		fmt.Printf("photo engine gRPC server running on %s\n", cfg.ListenAddr)
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
