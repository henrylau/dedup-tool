package core

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/kalafut/imohash"
)

type Scanner struct {
	Path    []string
	Storage Storage
	Logger  func(message string)
	Context context.Context
}

func (s *Scanner) Scan() error {
	if s.Context == nil {
		s.Context = context.Background()
	}
	hasher := imohash.New()

	for _, path := range s.Path {
		root, err := os.OpenRoot(path)
		if err != nil {
			return fmt.Errorf("failed to open root directory %s: %w", path, err)
		}

		err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
			select {
			case <-s.Context.Done():
				return s.Context.Err()
			default:
			}

			if err != nil {
				return err
			}
			if d.IsDir() || d.Name()[0] == '.' {
				return nil
			}

			f, err := root.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer f.Close()

			stats, err := f.Stat()
			if err != nil {
				return fmt.Errorf("failed to stat file %s: %w", path, err)
			}

			hash, err := getFileHash(f, hasher)
			if err != nil {
				return fmt.Errorf("failed to hash file %s: %w", path, err)
			}

			s.Storage.AddFile(&File{
				Path:    path,
				Hash:    hash,
				Size:    stats.Size(),
				ModTime: stats.ModTime(),
				Name:    stats.Name(),
			})

			if s.Logger != nil {
				s.Logger(fmt.Sprintf("scanned file %s: %s", path, hash))
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %w", path, err)
		}
	}
	return nil
}

// // ScanFolder recursively scans a directory and adds all files to storage.
// func ScanFolder(ctx context.Context, path string, storage Storage) error {
// 	dirFS := os.DirFS(path)

// 	return fs.WalkDir(dirFS, ".", func(path string, d fs.DirEntry, err error) error {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		default:
// 		}

// 		if err != nil {
// 			return err
// 		}
// 		if d.IsDir() || d.Name()[0] == '.' {
// 			return nil
// 		}

// 		f, err := dirFS.Open(path)
// 		if err != nil {
// 			return fmt.Errorf("failed to open file %s: %w", path, err)
// 		}
// 		defer f.Close()

// 		stats, err := f.Stat()
// 		if err != nil {
// 			return fmt.Errorf("failed to stat file %s: %w", path, err)
// 		}

// 		hash, err := FileHash(f, imohash.New())
// 		if err != nil {
// 			return fmt.Errorf("failed to hash file %s: %w", path, err)
// 		}

// 		storage.AddFile(&File{
// 			Path:    path,
// 			Hash:    hash,
// 			Size:    stats.Size(),
// 			ModTime: stats.ModTime(),
// 			Name:    stats.Name(),
// 		})

// 		return nil
// 	})
// }
