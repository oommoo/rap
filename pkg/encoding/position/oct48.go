package position

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/EliCDavis/vector"
	"github.com/recolude/rap/pkg/streams/position"
)

func floatBSTToBytes(value, start, duration float64, out []byte) {
	curValue := start + (duration / 2.0)
	increment := duration / 4.0

	for byteIndex := 0; byteIndex < len(out); byteIndex++ {

		// Clear whatever byte might be there
		out[byteIndex] = 0

		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			if value < curValue {
				curValue -= increment
			} else {
				out[byteIndex] = out[byteIndex] | (1 << bitIndex)
				curValue += increment
			}
			increment /= 2.0
		}
	}
}

func bytesToFloatBST(start, duration float64, in []byte) float64 {
	curValue := start + (duration / 2.0)
	increment := duration / 4.0

	for byteIndex := 0; byteIndex < len(in); byteIndex++ {
		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			if (in[byteIndex]>>byte(bitIndex))&1 == 1 {
				curValue += increment
			} else {
				curValue -= increment
			}

			increment /= 2.0
		}
	}

	return curValue
}

type OctCell int

// ORDER MATTERS: topBit << 2 | rightBit << 1 | forwardBit
const (
	TopRightForward OctCell = iota
	TopRightBackward
	TopLeftForward
	TopLeftBackward
	BottomRightForward
	BottomRightBackward
	BottomLeftForward
	BottomLeftBackward
)

func octCellsToBytes24(cells []OctCell, buffer []byte) {
	buffer[0] = 0
	buffer[1] = 0
	buffer[2] = 0

	buffer[0] = byte(cells[0])
	buffer[0] |= byte(cells[1]) << 3
	buffer[0] |= byte(cells[2]) << 6

	buffer[1] = byte(cells[2]) >> 2
	buffer[1] |= byte(cells[3]) << 1
	buffer[1] |= byte(cells[4]) << 4
	buffer[1] |= byte(cells[5]) << 7

	buffer[2] = byte(cells[5]) >> 1
	buffer[2] |= byte(cells[6]) << 2
	buffer[2] |= byte(cells[7]) << 5
}

func bytesToOctCells24(cells []OctCell, buffer []byte) {
	cells[0] = OctCell(buffer[0] & 0b111)
	cells[1] = OctCell((buffer[0] & 0b111000) >> 3)
	cells[2] = OctCell((buffer[0] >> 6) | ((buffer[1] & 0b1) << 2))
	cells[3] = OctCell((buffer[1] & 0b1110) >> 1)
	cells[4] = OctCell((buffer[1] & 0b1110000) >> 4)
	cells[5] = OctCell((buffer[1] >> 7) | ((buffer[2] & 0b11) << 1))
	cells[6] = OctCell((buffer[2] & 0b11100) >> 2)
	cells[7] = OctCell((buffer[2] & 0b11100000) >> 5)
}

func Vec3ToOctCells(v, min, max vector.Vector3, cells []OctCell) {
	center := min.Add(max).DivByConstant(2.0)
	crossSection := max.Sub(min)
	incrementX := crossSection.X() / 4.0
	incrementY := crossSection.Y() / 4.0
	incrementZ := crossSection.Z() / 4.0

	for cellIndex := 0; cellIndex < len(cells); cellIndex++ {
		topBit := 0
		newY := center.Y() + incrementY
		if v.Y() < center.Y() {
			topBit = 1
			newY = center.Y() - incrementY
		}

		rightBit := 0
		newX := center.X() + incrementX
		if v.X() < center.X() {
			rightBit = 1
			newX = center.X() - incrementX
		}

		forwardBit := 0
		newZ := center.Z() + incrementZ
		if v.Z() < center.Z() {
			forwardBit = 1
			newZ = center.Z() - incrementZ
		}
		cells[cellIndex] = OctCell(topBit<<2 | rightBit<<1 | forwardBit)
		center = vector.NewVector3(newX, newY, newZ)
		incrementX /= 2.0
		incrementY /= 2.0
		incrementZ /= 2.0
	}
}

func OctCellsToVec3(min, max vector.Vector3, cells []OctCell) vector.Vector3 {
	center := min.Add(max).DivByConstant(2.0)
	crossSection := max.Sub(min)
	incrementX := crossSection.X() / 4.0
	incrementY := crossSection.Y() / 4.0
	incrementZ := crossSection.Z() / 4.0

	for cellIndex := 0; cellIndex < len(cells); cellIndex++ {
		newY := center.Y() - incrementY
		if cells[cellIndex]&0b100 == 0 {
			newY = center.Y() + incrementY
		}

		newX := center.X() - incrementX
		if cells[cellIndex]&0b010 == 0 {
			newX = center.X() + incrementX
		}

		newZ := center.Z() - incrementZ
		if cells[cellIndex]&0b001 == 0 {
			newZ = center.Z() + incrementZ
		}
		center = vector.NewVector3(newX, newY, newZ)
		incrementX /= 2.0
		incrementY /= 2.0
		incrementZ /= 2.0
	}
	return center
}

func decodeOct24(streamData *bytes.Reader) ([]position.Capture, error) {
	var startTime float32
	err := binary.Read(streamData, binary.LittleEndian, &startTime)
	if err != nil {
		return nil, err
	}

	var duration float32
	err = binary.Read(streamData, binary.LittleEndian, &duration)
	if err != nil {
		return nil, err
	}

	var minX float32
	var minY float32
	var minZ float32
	var maxX float32
	var maxY float32
	var maxZ float32
	err = binary.Read(streamData, binary.LittleEndian, &minX)
	err = binary.Read(streamData, binary.LittleEndian, &minY)
	err = binary.Read(streamData, binary.LittleEndian, &minZ)
	err = binary.Read(streamData, binary.LittleEndian, &maxX)
	err = binary.Read(streamData, binary.LittleEndian, &maxY)
	err = binary.Read(streamData, binary.LittleEndian, &maxZ)
	min := vector.NewVector3(float64(minX), float64(minY), float64(minZ))
	max := vector.NewVector3(float64(maxX), float64(maxY), float64(maxZ))

	numCaptures, err := binary.ReadUvarint(streamData)
	if err != nil {
		return nil, err
	}

	captures := make([]position.Capture, numCaptures)
	timeBuffer := make([]byte, 2)
	octBuffer := make([]OctCell, 8)
	octBytesBuffer := make([]byte, 3)
	for i := 0; i < int(numCaptures); i++ {
		streamData.Read(timeBuffer)
		time := bytesToFloatBST(float64(startTime), float64(duration), timeBuffer)

		streamData.Read(octBytesBuffer)
		bytesToOctCells24(octBuffer, octBytesBuffer)
		v := OctCellsToVec3(min, max, octBuffer)

		captures[i] = position.NewCapture(time, v.X(), v.Y(), v.Z())
	}

	return captures, nil
}

func encodeOct24(captures []position.Capture) ([]byte, error) {
	streamData := new(bytes.Buffer)

	err := binary.Write(streamData, binary.LittleEndian, float32(captures[0].Time()))
	if err != nil {
		return nil, err
	}

	startingTime := math.Inf(1)
	endingTime := math.Inf(-1)

	min := vector.NewVector3(math.Inf(1), math.Inf(1), math.Inf(1))
	max := vector.NewVector3(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, capture := range captures {
		if capture.Time() < startingTime {
			startingTime = capture.Time()
		}
		if capture.Time() > endingTime {
			endingTime = capture.Time()
		}

		if capture.Position().X() > max.X() {
			max = max.SetX(capture.Position().X())
		}
		if capture.Position().Y() > max.Y() {
			max = max.SetY(capture.Position().Y())
		}
		if capture.Position().Z() > max.Z() {
			max = max.SetZ(capture.Position().Z())
		}

		if capture.Position().X() < min.X() {
			min = min.SetX(capture.Position().X())
		}
		if capture.Position().Y() < min.Y() {
			min = min.SetY(capture.Position().Y())
		}
		if capture.Position().Z() < min.Z() {
			min = min.SetZ(capture.Position().Z())
		}
	}

	duration := endingTime - startingTime

	err = binary.Write(streamData, binary.LittleEndian, float32(duration))
	if err != nil {
		return nil, err
	}

	// Write min and max positions
	binary.Write(streamData, binary.LittleEndian, float32(min.X()))
	binary.Write(streamData, binary.LittleEndian, float32(min.Y()))
	binary.Write(streamData, binary.LittleEndian, float32(min.Z()))
	binary.Write(streamData, binary.LittleEndian, float32(max.X()))
	binary.Write(streamData, binary.LittleEndian, float32(max.Y()))
	binary.Write(streamData, binary.LittleEndian, float32(max.Z()))

	// Write number of captures
	buf := make([]byte, 8)
	size := binary.PutUvarint(buf, uint64(len(captures)))
	streamData.Write(buf[:size])

	timeBuffer := make([]byte, 2)
	octBuffer := make([]OctCell, 8)
	octByteBuffer := make([]byte, 3)
	for _, capture := range captures {

		// Write Time
		floatBSTToBytes(capture.Time(), startingTime, duration, timeBuffer)
		_, err := streamData.Write(timeBuffer)
		if err != nil {
			return nil, err
		}

		// Write position
		Vec3ToOctCells(capture.Position(), min, max, octBuffer)
		octCellsToBytes24(octBuffer, octByteBuffer)
		_, err = streamData.Write(octByteBuffer)
		if err != nil {
			return nil, err
		}
	}

	return streamData.Bytes(), nil
}
