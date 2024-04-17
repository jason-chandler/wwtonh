package main

import (
	"encoding/binary"
	"io"
	"os"
	"slices"
)

// Most of the following is borrowed from IcySon55's MSBTEditor
// github.com/IcySon55/3DLandMSBTEditor
type EncodingByte byte

const (
	UTF8Byte EncodingByte = 0x00
	UnicodeByte EncodingByte = 0x01
)

type Encoding int64

const (
	UTF8 Encoding = iota
	Unicode
)

const expectedIdentifier = "MsgStdBn"

type Header struct {
	Identifier string // MsgStdBn
	ByteOrder []byte
	Unknown1 uint16 // Always 0x0000
	EncodingByte EncodingByte
	Unknown2 byte // Always 0x03
	NumberOfSections uint16
	Unknown3 uint16 // Always 0x0000
	FileSize uint32
	Unknown4 []byte // Always 0x0000 0000 0000 0000 0000
	FileSizeOffset int64
	Endianness binary.ByteOrder
}

type Section struct {
	Identifier string
	SectionSize uint32
	Padding1 []byte // Always 0x0000 0000
}

type Entry struct  {
	Value []byte
	Index uint32
}

type Group struct {
	NumberOfLabels uint32
	Offset uint32
}

type LBL1 struct {
	*Section
	NumberOfGroups uint32
	Groups []*Group
	Labels []*Label
}

type Label struct {
	*Entry
	Index uint32
	Length byte
	Name string
	Checksum uint32
}

type NLI1 struct {
	*Section
	Unknown2 []byte
}

type ATO1 struct {
	*Section
	Unknown2 []byte
}

type ATR1 struct {
	*Section
	NumberOfAttributes uint32
	Unknown2 []byte
}

type TSY1 struct {
	*Section
	Unknown2 []byte
}

type TXT2 struct {
	*Section
	NumberOfStrings uint32
	Strings []*Entry
	OriginalStrings []*Entry
}

const PaddingChar byte = 0xAB
const LabelMaxLength uint32 = 64


type MSBT struct {
	File []byte
	Header *Header
	Lbl1 *LBL1
	Nli1 *NLI1
	Ato1 *ATO1
	Atr1 *ATR1
	Tsy1 *TSY1
	Txt2 *TXT2
	Encoding Encoding
	SectionOrder []string
	HasLabels bool
}

func NewMSBT(filename string) *MSBT {
	f,err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	header := makeHeader(f)
	if header.FileSize != fileSize(filename) {
		panic("Written filesize does not match actual filesize.")
	}
	var lbl1 *LBL1
	var nli1 *NLI1
	var ato1 *ATO1
	var atr1 *ATR1
	var tsy1 *TSY1
	var txt2 *TXT2
	hasLabels := false
	
	// LabelFilter := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	sectionOrder := make([]string, 0)
	for _ = range header.NumberOfSections {
		sec := string(peekNBytes(f, 4))
		switch sec {
		case "LBL1":
			var labelsFound bool
			lbl1,labelsFound = readLbl1(f, header)
			if labelsFound {
				hasLabels = true
			}
			sectionOrder = append(sectionOrder, sec)
		case "NLI1":
			nli1 = readNli1(f, header)
			sectionOrder = append(sectionOrder, sec)
		case "ATO1":
			ato1 = readAto1(f, header)
			sectionOrder = append(sectionOrder, sec)
		case "ATR1":
			atr1 = readAtr1(f, header)
			sectionOrder = append(sectionOrder, sec)
		case "TSY1":
			tsy1 = readTsy1(f, header)
			sectionOrder = append(sectionOrder, sec)
		case "TXT2":
			txt2 = readTxt2(f, header)
			for _,lbl := range lbl1.Labels {
				lbl.Entry = txt2.Strings[lbl.Index]
			}
			sectionOrder = append(sectionOrder, sec)
		default:
			panic("Invalid section found: " + sec)
		}
	}

	var encoding Encoding
	if header.EncodingByte == UTF8Byte {
		encoding = UTF8
	} else {
		encoding = Unicode
	}
	return &MSBT {
		Header: header,
		Lbl1: lbl1,
		Nli1: nli1,
		Ato1: ato1,
		Atr1: atr1,
		Tsy1: tsy1,
		Txt2: txt2,
		Encoding: encoding,
		SectionOrder: sectionOrder,
		HasLabels: hasLabels,
	}
}

func readNBytes(f *os.File, n int64) []byte {
	buf := make([]byte, n)
	f.Read(buf)
	return buf
}

func peekNBytes(f *os.File, n int64) []byte {
	pos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}

	peekVal := readNBytes(f, n)
	_, err = f.Seek(pos, io.SeekStart)
	if err != nil {
		panic(err)
	}

	return peekVal
}

func makeHeader(f *os.File) *Header {
	identifBuf := readNBytes(f, 8)
	identifier := string(identifBuf[:])

	if identifier != expectedIdentifier {
		panic("Incorrect file type read as MSBT")
	}

	byteOrder := readNBytes(f, 2)
	var endianness binary.ByteOrder
	if byteOrder[0] > byteOrder[1] {
		endianness = binary.LittleEndian
	} else {
		endianness = binary.BigEndian
	}

	header := &Header{
		Identifier: identifier,
		ByteOrder: byteOrder,
		Endianness: endianness,
	}

	binary.Read(f, endianness, &header.Unknown1)
	binary.Read(f, endianness, &header.EncodingByte)
	binary.Read(f, endianness, &header.Unknown2)
	binary.Read(f, endianness, &header.NumberOfSections)
	binary.Read(f, endianness, &header.Unknown3)
	header.FileSizeOffset = currentSeek(f)
	binary.Read(f, endianness, &header.FileSize)
	header.Unknown4 = readNBytes(f, 10)

	return header
}

func currentSeek(f *os.File) int64 {
	var seekErr error
	curSeek,seekErr := f.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		panic(seekErr)
	}

	return curSeek
}

func fileSize(f string) uint32 {
	fileInfo, err := os.Stat(f)
	if err != nil {
		panic(err)
	}

	return uint32(fileInfo.Size())
}

func readLbl1(f *os.File, header *Header) (*LBL1,bool) {
	endianness := header.Endianness
	hasLabels := false
	lbl1 := &LBL1{ Section: &Section{} }
	lbl1.Identifier = string(readNBytes(f, 4))
	binary.Read(f, endianness, &lbl1.SectionSize)
	lbl1.Padding1 = readNBytes(f, 8)
	labelStart := currentSeek(f)
	binary.Read(f, endianness, &lbl1.NumberOfGroups)
	lbl1.Groups = make([]*Group, 0)
	for _ = range lbl1.NumberOfGroups {
		group := &Group{}
		binary.Read(f, endianness, &group.NumberOfLabels)
		binary.Read(f, endianness, &group.Offset)
		lbl1.Groups = append(lbl1.Groups, group)
	}

	lbl1.Labels = make([]*Label, 0)
	for _,group := range lbl1.Groups {
		f.Seek(labelStart + int64(group.Offset), 0)

		for _ = range group.NumberOfLabels {
			label := &Label{}
			binary.Read(f, endianness, &label.Length)
			label.Name = string(readNBytes(f, int64(label.Length)))
			binary.Read(f, endianness, &label.Index)
			label.Checksum = uint32(slices.Index(lbl1.Groups, group))
			lbl1.Labels = append(lbl1.Labels, label)
		}
	}

	if len(lbl1.Labels) > 0 {
		hasLabels = true
	}

	seekPastPadding(f)
	return lbl1, hasLabels
}

func seekPastPadding(f *os.File) {
	rem := currentSeek(f) % 16
	if rem > 0 {
		f.Seek(16 - rem, io.SeekCurrent)
	}
}

func readNli1(f *os.File, header *Header) *NLI1 {
	endianness := header.Endianness
	nli1 := &NLI1{}
	nli1.Identifier = string(readNBytes(f, 4))
	binary.Read(f, endianness, &nli1.SectionSize)
	nli1.Padding1 = readNBytes(f, 8)
	nli1.Unknown2 = readNBytes(f, int64(nli1.SectionSize))

	return nli1
}

func readAto1(f *os.File, header *Header) *ATO1 {
	endianness := header.Endianness
	ato1 := &ATO1{}
	ato1.Identifier = string(readNBytes(f, 4))
	binary.Read(f, endianness, &ato1.SectionSize)
	ato1.Padding1 = readNBytes(f, 8)
	ato1.Unknown2 = readNBytes(f, int64(ato1.SectionSize))

	return ato1
}

func readAtr1(f *os.File, header *Header) *ATR1 {
	endianness := header.Endianness
	atr1 := &ATR1{ Section: &Section{} }
	atr1.Identifier = string(readNBytes(f, 4))
	binary.Read(f, endianness, &atr1.SectionSize)
	atr1.Padding1 = readNBytes(f, 8)
	binary.Read(f, endianness, &atr1.NumberOfAttributes)
	atr1.Unknown2 = readNBytes(f, int64(atr1.SectionSize))
	seekPastPadding(f)

	return atr1
}

func readTsy1(f *os.File, header *Header) *TSY1 {
	endianness := header.Endianness
	tsy1 := &TSY1{ Section: &Section{} }
	tsy1.Identifier = string(readNBytes(f, 4))
	binary.Read(f, endianness, &tsy1.SectionSize)
	tsy1.Padding1 = readNBytes(f, 8)
	tsy1.Unknown2 = readNBytes(f, int64(tsy1.SectionSize))
	seekPastPadding(f)

	return tsy1
}

func readTxt2(f *os.File, header *Header) *TXT2 {
	endianness := header.Endianness
	txt2 := &TXT2{ Section: &Section{} }
	txt2.Identifier = string(readNBytes(f, 4))
	binary.Read(f, endianness, &txt2.SectionSize)
	txt2.Padding1 = readNBytes(f, 8)
	startStrings := currentSeek(f)
	binary.Read(f, endianness, &txt2.NumberOfStrings)
	offsets := make([]uint32, 0)

	for _ = range txt2.NumberOfStrings {
		var offset uint32
		binary.Read(f, endianness, &offset)
		offsets = append(offsets, offset)
	}

	for i := range txt2.NumberOfStrings {
		var nextOffset uint32
		if i + 1 < uint32(len(offsets)) {
			nextOffset = uint32(startStrings) + offsets[i + 1]
		} else {
			nextOffset = uint32(startStrings) + txt2.SectionSize
		}

		f.Seek(startStrings + int64(offsets[i]), io.SeekStart)

		result := make([]byte, 0)

		for {
			j := uint32(currentSeek(f))
			if j < nextOffset && j < header.FileSize {
				if(header.EncodingByte == UTF8Byte) {
					result = append(result, readNBytes(f, 1)[0])
				} else {
					var unibytes [2]byte
					binary.Read(f, endianness, &unibytes)
					result = append(result, unibytes[:]...)
				}
			} else {
				break
			}
		}

		entry := &Entry{
			Value: result,
			Index: i,
		}
		txt2.Strings = append(txt2.Strings, entry)
	}

	seekPastPadding(f)
	return txt2
}
