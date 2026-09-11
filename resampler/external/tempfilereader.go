package external

import "os"

type tempFileReader struct {
	f *os.File
}

func openTemp(name string) (*tempFileReader, error) {
	f, err := os.Open(name)
	return &tempFileReader{f: f}, err
}

func (r *tempFileReader) Read(p []byte) (int, error) {
	return r.f.Read(p)
}

func (r *tempFileReader) Close() error {
	if err := r.f.Close(); err != nil {
		return err
	}
	if err := os.Remove(r.f.Name()); err != nil {
		return err
	}
	return nil
}
