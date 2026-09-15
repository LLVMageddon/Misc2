package storage

import (
	"errors"
	"sync"
)

const (
	poolSize uint64 = 100
)

type BufferPoolManager struct {
	size        uint64
	frameData   []*Buffer
	pageTable   map[uint64]int
	emptyFrame  []int
	diskManager *DiskManager
	evictor     EvictionPolicy
	mu          sync.Mutex
}

type Buffer struct {
	pageId   uint64
	contents *SlottedPage
	// contents [PageSize]*byte
	pinCount uint16
	dirty    bool
	mu       sync.Mutex
}

func (b *Buffer) Reset() {
	b.pageId = 0
	b.contents = &SlottedPage{}
	b.pinCount = 0
	b.dirty = false
}

type EvictionPolicy interface {
	CandidateFrame() (uint64, bool)
	RemoveFrame(pFrameId uint64)
	RecordFrameAccess(pFrameId int)
}

func NewBufferPoolManager(pDiskManager *DiskManager) (*BufferPoolManager, error) {
	bpm := &BufferPoolManager{
		size:        0,
		frameData:   make([]*Buffer, 0, poolSize),
		pageTable:   make(map[uint64]int),
		emptyFrame:  make([]int, 0, poolSize),
		diskManager: pDiskManager,
		evictor:     nil,
	}
	for count := 0; count < int(poolSize); count++{
		bpm.emptyFrame = append(bpm.emptyFrame, count)
		bpm.frameData = append(bpm.frameData, &Buffer{contents: &SlottedPage{}})
	}
	return bpm, nil
}

func (bpm *BufferPoolManager) getFreeFrame() int {
	if len(bpm.emptyFrame) == 0 {
		return -1
	}

	idx := len(bpm.emptyFrame) - 1
	frameId := bpm.emptyFrame[idx]
	bpm.emptyFrame = bpm.emptyFrame[:idx]
	return frameId
}

// func (bpm *BufferPoolManager) NewPage() (*Buffer, error) {
func (bpm *BufferPoolManager) NewPage() (*SlottedPage, error) {
	// WIP:
	frameId := bpm.getFreeFrame()

	if frameId == -1 {
		frameId, ok := bpm.evictor.CandidateFrame() //TODO: Implement eviction policy
		if !ok {
			return nil, errors.New("no frames available for new page")
		}

		buffer := bpm.frameData[frameId]

		if buffer.pinCount > 0 {
			return nil, errors.New("eviction policy seleted a page with pinne hold")
		}

		if buffer.dirty {
			err := bpm.diskManager.WritePage(buffer.pageId, &buffer.contents.Data)
			if err != nil {
				return nil, err
			}
		}
		delete(bpm.pageTable, buffer.pageId)
		bpm.evictor.RemoveFrame(frameId)
		buffer.Reset()
	}

	pageId, err := bpm.diskManager.AllocatePage()
	if err != nil {
		return nil, err
	}

	buffer := bpm.frameData[frameId]
	err = bpm.diskManager.ReadPage(pageId, &buffer.contents.Data)
	if err != nil {
		return nil, err
	}

	buffer.Reset()
	buffer.pageId = pageId
	buffer.pinCount = 1
	buffer.dirty = false

	bpm.pageTable[pageId] = frameId
	bpm.evictor.RecordFrameAccess(frameId)

	// return buffer, nil
	return buffer.contents, nil
}

func (dpm *BufferPoolManager) FetchPage(pPageId uint64)(*SlottedPage, error){
	// TODO:
	panic("unimplemented")
}
