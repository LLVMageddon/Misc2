package storage

import (
	"errors"
	"math"
	"os"
	"sync"
)

type DiskManager struct {
	file     *os.File
	pageSize uint64 //NOTE: This is here incase page size changes based on versioning
	path     string

	mu sync.Mutex //NOTE: full concurrenct across every operation not need yet
}

func NewDiskManger(pPath string) (*DiskManager, error) {
	// REFACTOR:
	// DONE
	flag := os.O_RDWR | os.O_CREATE
	file, err := os.OpenFile(pPath, flag, 0644)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			file.Close()
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size()%int64(PageSize) != 0 {
		return nil, errors.New("corrupt database: file size is not page aligned")
		// return nil, errors.New("corrupt page detected")
	}

	return &DiskManager{
		file:     file,
		pageSize: PageSize,
		path:     pPath,
	}, nil
}

// HELPERS
func targetOffset(pPageId uint64) (int64, error) {
	if pPageId == 0 {
		return 0, errors.New("invalid pahe id: page IDs start at 1")
	}
	pPageId -= 1
	p := pPageId * PageSize

	if pPageId > uint64(math.MaxInt64)/PageSize {
		return 0, errors.New("page ID produces invalid file offset")
	}
	return int64(p), nil
}

//END OF HELPERS

func (dm *DiskManager) ReadPage(pPageId uint64, pPage *[PageSize]byte) error {
	// DONE
	if pPageId == 0 {
		return errors.New("invalid page Id")
	}

	if len(pPage) != int(dm.pageSize) {
		return errors.New("buffer size mismatch")
	}

	tempPage := make([]byte, PageSize)
	// copy(tempPage[:], pPage[:])
	pageoffset, err := targetOffset(pPageId)
	if err != nil {
		return err
	}

	pageSize, err := dm.file.ReadAt(tempPage[:], pageoffset)
	if err != nil {
		return err
	}

	if pageSize != int(dm.pageSize) {
		return errors.New("corrupt page detected")
	}

	copy(pPage[:], tempPage[:])
	return nil

}

func (dm *DiskManager) WritePage(pPageId uint64, pPage *[PageSize]byte) error {
	// DONE
	// NOTE: WritePage is intended to write the newly allocated page, rather than being a general overwrite operation.
	if pPageId == 0 {
		return errors.New("invalid page Id")
	}

	if len(pPage) != int(dm.pageSize) {
		return errors.New("corrupted page found")
	}

	fileStat, err := dm.file.Stat()
	if err != nil {
		return err
	}

	fileSize := fileStat.Size()
	if fileSize != int64(pPageId*PageSize) {
		return errors.New("unallocated space error")
	}

	pageoffset, err := targetOffset(pPageId)
	if err != nil {
		return err
	}

	var rollbackPage [PageSize]byte
	s, err := dm.file.ReadAt(rollbackPage[:], pageoffset)
	if err != nil {
		if s != PageSize {
			return err
		}
		return err
	}

	size, err := dm.file.WriteAt(pPage[:], pageoffset)
	if err != nil {
		return err
	}

	if size != int(PageSize) {
		dm.file.WriteAt(rollbackPage[:], pageoffset)
		// copy(pPage[:], rollbackPage[:])
		return errors.New("partial page write")
	}

	return nil
}

func (dm *DiskManager) AllocatePage() (uint64, error) {
	// DONE
	// dm.mu.Lock()
	// defer dm.mu.Unlock()

	info, err := dm.file.Stat()
	if err != nil {
		return 0, err
	}
	pageId := uint64(info.Size() / int64(dm.pageSize))
	page := make([]byte, dm.pageSize)

	pageoffset, err := targetOffset(pageId + 1)
	if err != nil {
		return 0, err
	}

	size, err := dm.file.WriteAt(page, pageoffset)
	if err != nil {
		return 0, err
	}

	if size != int(PageSize) {
		return 0, errors.New("partial page write")
	}

	return pageId + 1, nil
}

func (dm *DiskManager) Close() error {
	// DONE
	return dm.file.Close()
}
