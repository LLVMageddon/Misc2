package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	PageSize                  = 8192
	HeaderSize                = 32
	ReserverdSpaceSize        = 0
	DataSize                  = PageSize - HeaderSize //+ ReservedSpaceSize
	SlotOffsetSize            = 2
	SlotLengthSize            = 2
	SlotSize                  = 4
	PageMagic          uint32 = 0xDBDBDBDB
)

type SlottedPage struct {
	Data [PageSize]byte
}

const (
	OffsetMagic     = 0
	OffsetVersion   = 4
	OffsetFlags     = 6
	OffsetPageId    = 8
	OffsetFreeStart = 16
	OffsetFreeEnd   = 18
	OffsetSlotCount = 20
	OffsetChecksum  = 22
	OffsetReserved  = 26
	OffsetData      = HeaderSize // This is where the slots and and data is at
)

type slot struct {
	index  uint16
	offset uint16
	length uint16
}

type data struct {
	slots []slot
	data  [][]byte
}

type header struct {
	magic     uint32
	version   uint16
	flags     uint16
	pageId    uint64
	freeStart uint16
	freeEnd   uint16
	slotCount uint16
	checksum  uint32
	reserved  uint32
}

type page struct {
	header *header
	data   *data
}

// ERROR
func errHeaderOutOfBound(cause string, value uint) error {
	return fmt.Errorf("%s cause the header to be out of bounds\tHeader offset: %d\t\t %s offset: %d", cause, int(HeaderSize), cause, int(value))
}

//END OF ERRORS

// Helper Functions

func serialize(pPage *page) (*SlottedPage, error) {
	//TESTING: serialize page to slottedPage
	header, err := serializeHeader(pPage.header)
	if err != nil {
		return nil, err
	}

	data, err := serializeData(pPage.data)
	if err != nil {
		return nil, err
	}

	sp := &SlottedPage{}
	copy(sp.Data[0:HeaderSize], header)
	copy(sp.Data[HeaderSize:PageSize], data)

	return sp, nil
}

func serializeHeader(pHeader *header) ([]byte, error) {
	//TESTING: serialize header to []byte

	switch {
	case OffsetMagic > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetMagic", OffsetMagic)
	case OffsetVersion > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetVersion", OffsetVersion)
	case OffsetFlags > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetFlags", OffsetFlags)
	case OffsetPageId > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetPageId", OffsetPageId)
	case OffsetFreeStart > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetFreeStart", OffsetFreeStart)
	case OffsetFreeEnd > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetFreeEnd", OffsetFreeEnd)
	case OffsetSlotCount > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetSlotCount", OffsetSlotCount)
	case OffsetChecksum > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetChecksum", OffsetChecksum)
	case OffsetReserved > HeaderSize:
		return nil, errHeaderOutOfBound("OffsetReserved", OffsetReserved)
	}

	header := make([]byte, HeaderSize)
	binary.LittleEndian.PutUint32(header[OffsetMagic:], pHeader.magic)
	binary.LittleEndian.PutUint16(header[OffsetVersion:], pHeader.version)
	binary.LittleEndian.PutUint16(header[OffsetFlags:], pHeader.flags)
	binary.LittleEndian.PutUint64(header[OffsetPageId:], pHeader.pageId)
	binary.LittleEndian.PutUint16(header[OffsetFreeStart:], pHeader.freeStart)
	binary.LittleEndian.PutUint16(header[OffsetFreeEnd:], pHeader.freeEnd)
	binary.LittleEndian.PutUint16(header[OffsetSlotCount:], pHeader.slotCount)
	binary.LittleEndian.PutUint32(header[OffsetChecksum:], pHeader.checksum)
	binary.LittleEndian.PutUint32(header[OffsetReserved:], pHeader.reserved)

	return header, nil
}

func serializeData(pData *data) ([]byte, error) {
	//TESTING: serialize data to []byte

	slots := pData.slots
	data := pData.data
	if len(data) != len(slots) {
		return nil, errors.New("data corrupted: slots and raw data mismatch")
	}
	rawData := make([]byte, PageSize-OffsetData)

	// for x := 0; x < len(slots); x++{
	for idx, slot := range slots {
		if slot.offset > PageSize || slot.offset < HeaderSize {
			return nil, errors.New("data corrupted: data out of bounds for the allowed data bounds")
		}
		binary.LittleEndian.PutUint16(rawData[idx*SlotSize:], slots[idx].offset)
		binary.LittleEndian.PutUint16(rawData[idx*SlotSize+SlotLengthSize:], slots[idx].length)

		startOffset := (slots[idx].offset - slots[idx].length) - HeaderSize
		endOffset := (slots[idx].offset) - HeaderSize

		copy(rawData[startOffset:endOffset], data[idx])

	}

	return rawData, nil
}

func deserialize(pSlottedPage *SlottedPage) (*page, error) {
	//TESTING: deserialize slotted page to page struct
	headerRaw := pSlottedPage.Data[0:HeaderSize]
	dataRaw := pSlottedPage.Data[HeaderSize:PageSize]

	header, err := deserializeHeader(&headerRaw)
	if err != nil {
		return nil, err
	}

	data, err := deserializeData(&dataRaw)
	if err != nil {
		return nil, err
	}

	page := &page{
		header: header,
		data:   data,
	}
	return page, nil
}

func deserializeHeader(pHeader *[]byte) (*header, error) {
	//TESTING: deserialize header to header struct
	header := &header{
		magic:     binary.LittleEndian.Uint32((*pHeader)[OffsetMagic:]),
		version:   binary.LittleEndian.Uint16((*pHeader)[OffsetVersion:]),
		flags:     binary.LittleEndian.Uint16((*pHeader)[OffsetFlags:]),
		pageId:    binary.LittleEndian.Uint64((*pHeader)[OffsetPageId:]),
		freeStart: binary.LittleEndian.Uint16((*pHeader)[OffsetFreeStart:]),
		freeEnd:   binary.LittleEndian.Uint16((*pHeader)[OffsetFreeEnd:]),
		slotCount: binary.LittleEndian.Uint16((*pHeader)[OffsetSlotCount:]),
		checksum:  binary.LittleEndian.Uint32((*pHeader)[OffsetChecksum:]),
		reserved:  binary.LittleEndian.Uint32((*pHeader)[OffsetReserved:]),
	}

	return header, nil
}

func deserializeData(pData *[]byte) (*data, error) {
	//TODO: deserialize data to data struct
	//NOTE: I will combine slots and data
	return nil, nil
}

// End of Helper functions

func NewSlottedPage(pPageId uint64) (*SlottedPage, error) {
	//TESTING: Create new slotted page

	header := &header{
		magic:     PageMagic,
		version:   1,
		flags:     0,
		pageId:    pPageId,
		freeStart: OffsetData,
		freeEnd:   PageSize,
		checksum:  0,
		reserved:  0,
	}

	data := &data{
		slots: nil,
		data:  nil,
	}

	page := &page{
		header: header,
		data:   data,
	}

	sp, err := serialize(page)

	if err != nil {
		return nil, err
	}

	return sp, nil
}
