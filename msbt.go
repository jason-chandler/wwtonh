package main

import (
	"encoding/binary"
	"io"
	"os"
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

type Entry interface  {
	Value() []byte
	SetValue(value []byte)
	Index() uint32
	SetIndex(index uint32)
}

type Group struct {
	NumberOfLabels uint32
	Offset uint32
}

type LBL1 struct {
	*Section
	NumberOfGroups uint32
	Groups []*Group
	Labels []*Entry
}

type Label struct {
	Entry
	Index uint32
	Length uint32
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
	Head *Header
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

func NewMSBT(filename string) {
	f,err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	header := makeHeader(f)
	if header.FileSize == fileSize(filename) {
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
			lbl1 = readLbl1(f, header.Endianness)
			sectionOrder = append(sectionOrder, sec)
		case "NLI1":
			sectionOrder = append(sectionOrder, sec)
		case "ATO1":
			sectionOrder = append(sectionOrder, sec)
		case "ATR1":
			sectionOrder = append(sectionOrder, sec)
		case "TSY1":
			sectionOrder = append(sectionOrder, sec)
		case "TXT2":
			sectionOrder = append(sectionOrder, sec)
		default:
			panic("Invalid section found: " + sec)
		}
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
	}

	binary.Read(f, endianness, header.Unknown1)
	binary.Read(f, endianness, header.EncodingByte)
	binary.Read(f, endianness, header.Unknown2)
	binary.Read(f, endianness, header.NumberOfSections)
	binary.Read(f, endianness, header.Unknown3)
	header.FileSizeOffset = currentSeek(f)
	binary.Read(f, endianness, header.FileSize)
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

func readLbl1(f *os.File, endianness binary.ByteOrder) *LBL1 {
	lbl1 := &LBL1{}
	binary.Read(f, endianness, lbl1.Identifier)
	binary.Read(f, endianness, lbl1.SectionSize)
	lbl1.Padding1 = readNBytes(f, 8)
	labelStart := currentSeek(f)
	binary.Read(f, endianness, lbl1.NumberOfGroups)
	lbl1.Groups = make([]*Group, 0)
	for _ = range lbl1.NumberOfGroups {
		group := &Group{}
		binary.Read(f, endianness, group.NumberOfLabels)
		binary.Read(f, endianness, group.Offset)
		lbl1.Groups = append(lbl1.Groups, group)
	}

	for _,group := range lbl1.Groups {
		f.Seek(labelStart + int64(group.Offset), 0)
		
	}
	return lbl1
}
