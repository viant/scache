package scache

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"syscall"
)

const filePermission = 0644

type mmap struct {
	location string
	file     *os.File
	size     int
}

func (m *mmap) close() error {
	return m.file.Close()
}

func (m *mmap) open() error {
	var openingFlag = os.O_RDWR
	if _, err := os.Stat(m.location); os.IsNotExist(err) {
		openingFlag = openingFlag | os.O_CREATE
	}
	var err error
	if m.file, err = os.OpenFile(m.location, openingFlag, filePermission); err != nil {
		return errors.Wrapf(err, "failed to create file: %v", m.location)
	}
	return m.allocate()
}

func (m *mmap) assign(offset int64, target *[]byte) error {
	buffer, err := syscall.Mmap(int(m.file.Fd()), offset, m.size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return errors.Wrapf(err, "failed to map memory %v", m.location)
	}
	*target = buffer
	return nil
}

func (m *mmap) allocate() error {
	info, err := os.Stat(m.location)
	if err != nil {
		return err
	}
	if info.Size() < int64(m.size) {
		_, err := m.file.Seek(int64(m.size-1), 0)
		if err != nil {
			return fmt.Errorf("Failed to seek file %v", err)
		}
		_, err = m.file.Write([]byte{0})
		if err != nil {
			return errors.Wrapf(err, "failed to resize %v", m.location)
		}
	}
	return nil
}

func newMmap(location string, size int) *mmap {
	return &mmap{
		location: location,
		size:     size,
	}
}
