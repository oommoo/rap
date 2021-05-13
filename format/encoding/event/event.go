package event

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/recolude/rap/format"
	"github.com/recolude/rap/format/streams/event"
	rapbinary "github.com/recolude/rap/internal/io/binary"
)

type StorageTechnique int

const (
	// Raw64 encodes time with 64 bit precision
	Raw64 StorageTechnique = iota

	// Raw32 encodes time with 32 bit precision
	Raw32
)

type Encoder struct {
	technique StorageTechnique
}

func NewEncoder(technique StorageTechnique) Encoder {
	return Encoder{technique}
}

func (p Encoder) Accepts(stream format.CaptureStream) bool {
	return stream.Signature() == "recolude.event"
}

func (p Encoder) Signature() string {
	return "recolude.event"
}

func (p Encoder) Version() uint {
	return 0
}

func (p Encoder) Encode(streams []format.CaptureStream) ([]byte, [][]byte, error) {
	eventNamesSet := make(map[string]int)
	eventKeysSet := make(map[string]int)

	streamDataBuffers := make([]bytes.Buffer, len(streams))
	for bufferIndex, stream := range streams {

		// Write Stream Name
		streamDataBuffers[bufferIndex].Write(rapbinary.StringToBytes(stream.Name()))

		// Write technique
		streamDataBuffers[bufferIndex].WriteByte(byte(p.technique))

		// Write Num Captures
		numCaptures := make([]byte, 4)
		read := binary.PutUvarint(numCaptures, uint64(len(stream.Captures())))
		streamDataBuffers[bufferIndex].Write(numCaptures[:read])

		for _, c := range stream.Captures() {
			eventCapture, ok := c.(event.Capture)
			if !ok {
				return nil, nil, errors.New("capture is not of type event")
			}

			switch p.technique {
			case Raw64:
				binary.Write(&streamDataBuffers[bufferIndex], binary.LittleEndian, eventCapture.Time())
			case Raw32:
				binary.Write(&streamDataBuffers[bufferIndex], binary.LittleEndian, float32(eventCapture.Time()))
			}

			if _, ok := eventNamesSet[eventCapture.Name()]; !ok {
				eventNamesSet[eventCapture.Name()] = len(eventNamesSet)
			}

			nameIndex := make([]byte, 4)
			read := binary.PutUvarint(nameIndex, uint64(eventNamesSet[eventCapture.Name()]))
			streamDataBuffers[bufferIndex].Write(nameIndex[:read])

			allKeyIndxes := make([]uint, len(eventCapture.Metadata()))
			allValueStrings := make([]string, len(eventCapture.Metadata()))
			keyCount := 0
			for key, val := range eventCapture.Metadata() {
				if _, ok := eventKeysSet[key]; !ok {
					eventKeysSet[key] = len(eventKeysSet)
				}
				allKeyIndxes[keyCount] = uint(eventKeysSet[key])
				allValueStrings[keyCount] = val
				keyCount++
			}

			streamDataBuffers[bufferIndex].Write(rapbinary.UintArrayToBytes(allKeyIndxes))
			streamDataBuffers[bufferIndex].Write(rapbinary.StringArrayToBytes(allValueStrings))
		}
	}

	streamData := make([][]byte, len(streams))
	for i, buffer := range streamDataBuffers {
		streamData[i] = buffer.Bytes()
	}

	header := bytes.Buffer{}

	allNames := make([]string, len(eventNamesSet))
	for key, index := range eventNamesSet {
		allNames[index] = key
	}
	header.Write(rapbinary.StringArrayToBytes(allNames))

	allKeys := make([]string, len(eventKeysSet))
	for key, index := range eventKeysSet {
		allKeys[index] = key
	}
	header.Write(rapbinary.StringArrayToBytes(allKeys))

	return header.Bytes(), streamData, nil
}

func readHeader(header []byte) (names []string, metadataKeys []string, err error) {
	headerReader := bytes.NewBuffer(header)
	names, _, err = rapbinary.ReadStringArray(headerReader)
	metadataKeys, _, err = rapbinary.ReadStringArray(headerReader)
	return
}

func (p Encoder) Decode(header []byte, streamData []byte) (format.CaptureStream, error) {
	buf := bytes.NewBuffer(streamData)

	eventNames, metadataKeys, err := readHeader(header)
	if err != nil {
		return nil, err
	}

	// Read Name
	streamName, _, err := rapbinary.ReadString(buf)
	if err != nil {
		return nil, err
	}

	// Read Storage Technique
	typeByte, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	encodingTechnique := StorageTechnique(typeByte)

	// Read Num Captures
	numCaptures, err := binary.ReadUvarint(buf)
	if err != nil {
		return nil, err
	}

	captures := make([]event.Capture, numCaptures)
	for i := 0; i < int(numCaptures); i++ {
		var time float64

		switch encodingTechnique {
		case Raw64:
			binary.Read(buf, binary.LittleEndian, &time)

		case Raw32:
			var time32 float32
			binary.Read(buf, binary.LittleEndian, &time32)
			time = float64(time32)
		}

		eventNameIndex, err := binary.ReadUvarint(buf)
		if err != nil {
			return nil, err
		}

		metadataIndeces, _, err := rapbinary.ReadUintArray(buf)
		if err != nil {
			return nil, err
		}

		metadataValues, _, err := rapbinary.ReadStringArray(buf)
		if err != nil {
			return nil, err
		}

		metadata := make(map[string]string)
		for metadataIndex := 0; metadataIndex < len(metadataIndeces); metadataIndex++ {
			metadata[metadataKeys[metadataIndeces[metadataIndex]]] = metadataValues[metadataIndex]
		}
		captures[i] = event.NewCapture(time, eventNames[int(eventNameIndex)], metadata)
	}

	return event.NewStream(streamName, captures), nil
}