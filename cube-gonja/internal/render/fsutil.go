package render

import (
	"fmt"
	"io"
	"os"
)

func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func CopyAtomic(dst string, src io.Reader, perm os.FileMode) error {
	b, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	return WriteAtomic(dst, b, perm)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func MkdirAll(path string) error {
	if Exists(path) {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}

func RemoveAllFiles(dir string) error {
	d, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range d {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(fmt.Sprintf("%s/%s", dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
