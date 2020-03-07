package util

import "io"

type rwc struct {
	r io.ReadCloser
}

func (r *rwc) Read(bs []byte) (int, error) {
	return r.r.Read(bs)
}

func (r *rwc) Write(bs []byte) (int, error) {
	return len(bs), nil
}

func (r *rwc) Close() error {
	return r.r.Close()
}

func NopRWCloser(r io.ReadCloser) io.ReadWriteCloser {
	return &rwc{r}
}
