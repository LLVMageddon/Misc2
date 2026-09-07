package storage

//REFACTOR

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"slices"
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
	slots := pData.slots
	data := pData.data
	if len(data) != len(slots) {
		return nil, errors.New("data corrupted: slots and raw data mismatch")
	}
	rawData := make([]byte, PageSize-OffsetData)

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
	headerRaw := pSlottedPage.Data[0:HeaderSize]
	dataRaw := pSlottedPage.Data[HeaderSize:PageSize]

	header, err := deserializeHeader(headerRaw)
	if err != nil {
		return nil, err
	}

	data, err := deserializeData(header.slotCount, dataRaw)
	if err != nil {
		return nil, err
	}

	page := &page{
		header: header,
		data:   data,
	}
	return page, nil
}

func deserializeHeader(pHeader []byte) (*header, error) {
	header := &header{
		magic:     binary.LittleEndian.Uint32(pHeader[OffsetMagic:]),
		version:   binary.LittleEndian.Uint16(pHeader[OffsetVersion:]),
		flags:     binary.LittleEndian.Uint16(pHeader[OffsetFlags:]),
		pageId:    binary.LittleEndian.Uint64(pHeader[OffsetPageId:]),
		freeStart: binary.LittleEndian.Uint16(pHeader[OffsetFreeStart:]),
		freeEnd:   binary.LittleEndian.Uint16(pHeader[OffsetFreeEnd:]),
		slotCount: binary.LittleEndian.Uint16(pHeader[OffsetSlotCount:]),
		checksum:  binary.LittleEndian.Uint32(pHeader[OffsetChecksum:]),
		reserved:  binary.LittleEndian.Uint32(pHeader[OffsetReserved:]),
	}

	return header, nil
}

func deserializeData(pSlotCount uint16, pData []byte) (*data, error) {
	slotCount := pSlotCount
	if (int(slotCount) * SlotSize) > len(pData){
		return nil, errors.New("corrupt page: slot dir exceeds page data")
	}
	slots := make([]slot, slotCount)
	// var rawData [][]byte
	rawData := make([][]byte, slotCount)

	for count := 0; uint16(count) < slotCount; count++ {

		slotOffset := count * SlotSize
		offset := binary.LittleEndian.Uint16(pData[slotOffset:])
		length := binary.LittleEndian.Uint16(pData[slotOffset+SlotLengthSize:])
		
		if length != 0{
			if offset < HeaderSize || offset > PageSize || length > (offset - HeaderSize){
				return nil, errors.New("corrupt page: slot record out of bounds")
			}

			start := offset - HeaderSize - length
			end := offset - HeaderSize

			if start < 0 || int(end) > len(pData) || start > end {
				return nil, errors.New("corrupt pahe: invalid record range")
			}

			record := make([]byte, length)
			copy(record, pData[start:end])
			rawData[count] = record


		}

		slots[count] = slot{
			index: uint16(count),
			offset: offset,
			length: length,
		}	
	}

	data := &data{
		slots: slots,
		data:  rawData,
	}

	return data, nil
}

func validateChecksum(pData [PageSize]byte) (bool, error) {

	checksum := binary.LittleEndian.Uint32(pData[OffsetChecksum:])
	cloneData := make([]byte, len(pData))
	copy(cloneData[:], pData[:])
	binary.LittleEndian.PutUint32(cloneData[OffsetChecksum:], 0)
	cloneDataChecksum := crc32.ChecksumIEEE(cloneData[:])

	if checksum != cloneDataChecksum {
		return false, errors.New("data corrupted: CRC mismatch")
	}

	return true, nil
}

func recalculateChecksum(rawData *[8192]byte) {
	binary.LittleEndian.PutUint32(rawData[OffsetChecksum:], 0)
	checksum := crc32.ChecksumIEEE(rawData[:])
	binary.LittleEndian.PutUint32(rawData[OffsetChecksum:], checksum)
}

// End of Helper functions

func NewSlottedPage(pPageId uint64) (*SlottedPage, error) {
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

	calculatedChecksum := crc32.ChecksumIEEE(sp.Data[:])
	binary.LittleEndian.PutUint32(sp.Data[OffsetChecksum:], calculatedChecksum)

	return sp, nil
}

func (sp *SlottedPage) getPage() (*page, error) {
	p, err := deserialize(sp)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (sp *SlottedPage) addRecord(pRecord []byte) (uint16, error) {
	//REFACTOR
	// TBD: Should record id start at 1 or 0?
	// NOTE: Per dev doc: you do not need to leave an artifical buffer between slots and record data. They can safely touch. The only padding you need to worry about is hardware/data alignment, ensuring records start on 4-byte or 8-byte boundaries.:set wrap
	validChecksum, err := validateChecksum(sp.Data)
	if err != nil && !validChecksum{
		return 0, err
	}

	if len(pRecord) > (PageSize - HeaderSize - SlotSize){
		return 0, errors.New("record is too large for a page")
	}

	recordSize := uint16(len(pRecord))
	// if recordSize == 0 {
	// 	return 0, nil
	// }

	rawData := &sp.Data

	slotCount := binary.LittleEndian.Uint16(rawData[OffsetSlotCount:])
	freeStart := binary.LittleEndian.Uint16(rawData[OffsetFreeStart:])
	freeEnd := binary.LittleEndian.Uint16(rawData[OffsetFreeEnd:])

	if freeStart > freeEnd || freeEnd > PageSize || freeStart < HeaderSize{
		return 0, errors.New("corrupt page: invalud free-space bounds")
	}

	freespace := freeEnd - freeStart

	for count := uint16(0); count < slotCount; count++ {
		slotOffset := HeaderSize + (count * SlotSize)
		length := binary.LittleEndian.Uint16(rawData[slotOffset+SlotLengthSize:])
		if length == 0 {
			// if freeEnd < recordSize || (freeEnd - recordSize) < freeStart{ //
			// if freespace <= recordSize {
				// if freespace < recordSize {
				// continue
				// break
			// }
			if length != 0{
				continue
			}
			if freespace < recordSize{
				break
			}

			newFreeEnd := freeEnd - recordSize

			copy(rawData[newFreeEnd:freeEnd], pRecord[:])
			
			binary.LittleEndian.PutUint16(rawData[slotOffset:], uint16(freeEnd))
			binary.LittleEndian.PutUint16(rawData[slotOffset+SlotLengthSize:], uint16(recordSize))
			binary.LittleEndian.PutUint16(rawData[OffsetFreeEnd:], newFreeEnd)
			recalculateChecksum(rawData)
			return count + 1, nil
		}
	}

	// if (freeEnd - freeStart) <= recordSize + SlotSize {
	if freespace <= recordSize+SlotSize {
		// if freespace < recordSize+SlotSize {
		// err := sp.compactPage()
		err := sp.compactPage()
		if err != nil {
			return 0, fmt.Errorf("error: there was an error compacting the page\t %v", err)
		}

		freeStart = binary.LittleEndian.Uint16(rawData[OffsetFreeStart:])
		freeEnd = binary.LittleEndian.Uint16(rawData[OffsetFreeEnd:])
		slotCount = binary.LittleEndian.Uint16(rawData[OffsetSlotCount:])
		if (freeEnd - (freeStart + SlotSize)) <= recordSize {
			// if (freeEnd - (freeStart + SlotSize)) < recordSize {
			return 0, errors.New("error: page is full")
		}
	}

	copy(rawData[freeEnd-recordSize:freeEnd], pRecord[:])
	binary.LittleEndian.PutUint16(rawData[HeaderSize+(slotCount*SlotSize):], uint16(freeEnd))
	binary.LittleEndian.PutUint16(rawData[HeaderSize+(slotCount*SlotSize)+SlotLengthSize:], uint16(recordSize))
	slotCount++
	binary.LittleEndian.PutUint16(rawData[OffsetSlotCount:], slotCount)
	binary.LittleEndian.PutUint16(rawData[OffsetFreeEnd:], uint16(freeEnd)-uint16(recordSize))
	binary.LittleEndian.PutUint16(rawData[OffsetFreeStart:], uint16(freeStart)+SlotSize)

	recalculateChecksum(rawData)

	return slotCount, nil //FIXME: you cant use the slotCount and recourd id?
}

func (sp *SlottedPage) compactPageOld() error {
	// REFACTOR
	// FIXME: Logic is wrong
	data := &sp.Data
	slotCount := binary.LittleEndian.Uint16(data[OffsetSlotCount:])

	if slotCount == 0 {
		return errors.New("error: slot count before compact is 0")
	}

	// slots := make([]slot, slotCount)
	slots := make([]slot, 0)
	// slotLengthOffset := OffsetFreeStart

	for idx := uint16(0); idx < slotCount; idx++ {
		slotLengthOffset := (idx * SlotSize) + HeaderSize
		length := binary.LittleEndian.Uint16(data[slotLengthOffset+SlotOffsetSize:])
		if length != 0 {
			slots = append(slots, slot{
				index:  idx,
				offset: binary.LittleEndian.Uint16(data[slotLengthOffset:]),
				length: binary.LittleEndian.Uint16(data[slotLengthOffset+SlotOffsetSize:]),
			})
		}
	}

	slices.SortFunc(slots, func(a, b slot) int {
		// NOTE: Sorting slots based on data offset desc
		return cmp.Compare(b.offset, a.offset)
	})

	tempBuf := make([]byte, PageSize)
	newFreeStart := HeaderSize
	newFreeEnd := PageSize

	for idx := 0; idx < len(slots); idx++ {
		slotOffset := HeaderSize + (idx * SlotSize)
		binary.LittleEndian.PutUint16(tempBuf[slotOffset:], slots[idx].offset)
		binary.LittleEndian.PutUint16(tempBuf[slotOffset+SlotLengthSize:], slots[idx].length)
		newFreeStart += SlotSize
		newFreeEnd -= int(slots[idx].length)
	}

	copy(data[HeaderSize:PageSize], tempBuf[HeaderSize:PageSize])
	binary.LittleEndian.PutUint16(data[OffsetFreeStart:], uint16(newFreeStart))
	binary.LittleEndian.PutUint16(data[OffsetFreeEnd:], uint16(newFreeEnd))
	binary.LittleEndian.PutUint16(data[OffsetSlotCount:], uint16(len(slots)))

	recalculateChecksum(data)

	return nil
}

func (sp *SlottedPage) compactPage() error{
	//WIP:
	//FIXME: Logic is wrong
	//TBD: Back to docs, for proper understanding, not workd of mouth.
	data := &sp.Data
	slotCount := binary.LittleEndian.Uint16(data[OffsetSlotCount:])

	if slotCount == 0{
		return nil
	}

	type liveSlot struct{
		index uint16
		length uint16
		bytes []byte
	}

	slots := make([]liveSlot, 0, slotCount)

	for idx := uint16(0); idx < slotCount; idx ++{
		slotOffset := HeaderSize + (idx * SlotSize)
		offset := binary.LittleEndian.Uint16(data[slotOffset:])
		length := binary.LittleEndian.Uint16(data[slotOffset+SlotOffsetSize:])

		if length == 0{
			continue
		}

		if offset < HeaderSize || offset > PageSize || length > (offset - HeaderSize){
			return errors.New("corrup page: invalid slot bounds")
		}

		record := make([]byte, length)
		copy(record, data[offset - length: offset])

		slots = append(slots, liveSlot{
			index : idx,
			length: length,
			bytes: record,
		})
	}

	freeEnd := uint16(PageSize)
	
	// for _, s := range slots{
	for idx := 0; idx < len(slots); idx++{
		s := slots[idx]
		copy(data[freeEnd - s.length:freeEnd], s.bytes)
		
		// slotOffset := HeaderSize + (s.index * SlotSize)
		slotOffset := HeaderSize + (idx * SlotSize)
		binary.LittleEndian.PutUint16(data[slotOffset:], freeEnd - s.length)
		binary.LittleEndian.PutUint16(data[slotOffset+SlotLengthSize:], s.length)
		freeEnd -= s.length
	}

	freeStart := HeaderSize + (slotCount * SlotSize)
	if freeStart > freeEnd{
		return errors.New("corrupt page: records overlap slot dir")
	}

	binary.LittleEndian.PutUint16(data[OffsetFreeStart:], freeStart)
	binary.LittleEndian.PutUint16(data[OffsetFreeEnd:], freeEnd)

	recalculateChecksum(data)
	return nil

}

func (sp *SlottedPage) removeRecord(pRecordId uint16) (bool, error) {
	// DONE
	// TBD: Should record id start at 1 or 0?
	validChecksum, err := validateChecksum(sp.Data)
	if err != nil && !validChecksum{
		return false, err
	}

	data := &sp.Data
	slotCount := binary.LittleEndian.Uint16(data[OffsetSlotCount:])
	if slotCount == 0 {
		return false, errors.New("error record does not exist")
	}
	if slotCount < pRecordId {
		return false, errors.New("error: record does not exist")
	}
	pRecordId -= 1

	length := binary.LittleEndian.Uint16(data[HeaderSize+(SlotSize*pRecordId)+SlotLengthSize:])
	if length == 0{
		return false, errors.New("error: record already deleted")
	}

	binary.LittleEndian.PutUint16(data[HeaderSize+(SlotSize*pRecordId)+SlotLengthSize:], 0)
	recalculateChecksum(data)

	return true, nil
}

func (sp *SlottedPage) getRecord(pRecordId uint16) ([]byte, error) {
	// DONE
	// TBD: Should record id start at 1 or 0?
	data := &sp.Data

	validChecksum, err := validateChecksum(sp.Data)
	if err != nil && !validChecksum {
		return nil, err
	}
	
	slotCount := binary.LittleEndian.Uint16(data[OffsetSlotCount:])
	if slotCount == 0 {
		return nil, errors.New("error record does not exist")
	}
	if slotCount < pRecordId {
		return nil, errors.New("error: record does not exist")
	}

	pRecordId -= 1
	slotOffset := binary.LittleEndian.Uint16(data[HeaderSize+(SlotSize*pRecordId):])
	slotLength := binary.LittleEndian.Uint16(data[HeaderSize+(SlotSize*pRecordId)+SlotLengthSize:])
	
	if slotLength == 0 {
		return nil, errors.New("error: record does not exist")
	}

	returnData := make([]byte, slotLength)
	copy(returnData[:], data[slotOffset-slotLength:slotOffset])

	return returnData, nil
}

func (sp *SlottedPage) updateRecord(pRecordId uint16, pRecord []byte) (uint16, error) {
	// TODO: implement this record update function 
	// TBD: Is this really needed? Its go the require alot of shifting of data.
	panic("unimplemented")
}
