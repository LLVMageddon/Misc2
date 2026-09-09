package storage

type DiskManager struct {
	// TODO: add implementation detail
}

func NewDiskManger(pPath string) (*DiskManager, error) {
	// TODO: add implementation detail
	panic("unimplemented")
}

func (dm *DiskManager) ReadPage(pPageId uint64, pPage *[PageSize]byte) error {
	// TODO: add implementation detail
	panic("unimplemented")
}

func (dm *DiskManager) WritePage(pPageId uint64, pPage *[PageSize]byte) error {
	// TODO: add implementation detail
	panic("unimplemented")
}

func (dm *DiskManager) AllocatePage() (uint64, error) {
	// TODO: add implementation detail
	panic("unimplemented")
}

func (dm *DiskManager) Close() error {
	// TODO: add implementation detail
	panic("unimplemted")
}
