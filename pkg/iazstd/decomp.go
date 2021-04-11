package iazstd

// LICENSE NOTE.. THIS CODE IS BORROWED FROM rewby@#archiveteam-bs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/valyala/gozstd"
	"io"
)

func IsPlainZstdFile(r *bufio.Reader) (bool, error) {
	magic, err := r.Peek(4)
	if err != nil {
		return false, err
	}
	if bytes.Equal(magic, []byte{0x28, 0xB5, 0x2F, 0xFD}) {
		return true, nil
	}

	return false, nil
}

func IsDictZstdFile(r *bufio.Reader) (bool, error) {
	magic, err := r.Peek(4)
	if err != nil {
		return false, err
	}
	if bytes.Equal(magic, []byte{0x5D, 0x2A, 0x4D, 0x18}) {
		return true, nil
	}

	return false, nil
}

func NewZstdDictReader(r *bufio.Reader) (io.Reader, error) {
	//check for skippable frame
	magic, err := r.Peek(4)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(magic, []byte{0x5D, 0x2A, 0x4D, 0x18}) {
		return nil, errors.New("not a zstd file")
	}

	lenbytes := make([]byte, 8)
	n, err := io.ReadFull(r, lenbytes)
	if err != nil {
		return nil, err
	}
	if n != 8 {
		return nil, errors.New("not enough bytes")
	}

	length := binary.LittleEndian.Uint32(lenbytes[4:8])

	rawDict := make([]byte, length)
	n, err = io.ReadFull(r, rawDict)
	if err != nil {
		return nil, err
	}
	if uint32(n) != length {
		return nil, errors.New("not enough bytes")
	}

	var parsedDict *gozstd.DDict
	//Check if raw_dict is a zstd file.
	if bytes.Equal(rawDict[:4], []byte{0x28, 0xB5, 0x2F, 0xFD}) {
		rawDict, err = gozstd.Decompress(nil, rawDict)
		if err != nil {
			return nil, err
		}
	}
	parsedDict, err = gozstd.NewDDict(rawDict)
	if err != nil {
		return nil, err
	}
	//Check if parsed_dict is a dictionary.
	// if !bytes.Equal(parsed_dict[:4], []byte{0x37, 0xA4, 0x30, 0xEC}) {
	//	return nil, errors.New("not a dict")
	//}

	return gozstd.NewReaderDict(r, parsedDict), nil
}
