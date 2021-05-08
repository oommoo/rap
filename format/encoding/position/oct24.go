package position

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/EliCDavis/vector"
	"github.com/recolude/rap/format/streams/position"
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

func encodeOct24(captures []position.Capture) ([]byte, error) {
	streamData := new(bytes.Buffer)

	// Write number of captures
	buf := make([]byte, 8)
	size := binary.PutUvarint(buf, uint64(len(captures)))
	streamData.Write(buf[:size])

	if len(captures) == 0 {
		return streamData.Bytes(), nil
	}

	err := binary.Write(streamData, binary.LittleEndian, float32(captures[0].Time()))
	if err != nil {
		return nil, err
	}

	if len(captures) == 1 {
		err = binary.Write(streamData, binary.LittleEndian, float32(captures[0].Position().X()))
		if err != nil {
			return nil, err
		}
		err = binary.Write(streamData, binary.LittleEndian, float32(captures[0].Position().Y()))
		if err != nil {
			return nil, err
		}
		err = binary.Write(streamData, binary.LittleEndian, float32(captures[0].Position().Z()))
		return streamData.Bytes(), err
	}

	startingTime := math.Inf(1)
	endingTime := math.Inf(-1)
	maxTimeDifference := math.Inf(-1)

	min := vector.NewVector3(math.Inf(1), math.Inf(1), math.Inf(1))
	max := vector.NewVector3(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for i, capture := range captures {
		if capture.Time() < startingTime {
			startingTime = capture.Time()
		}
		if capture.Time() > endingTime {
			endingTime = capture.Time()
		}

		if i > 0 {
			timeDifference := capture.Time() - captures[i-1].Time()
			if timeDifference > maxTimeDifference {
				maxTimeDifference = timeDifference
			}

			distance := capture.Position().Sub(captures[i-1].Position())

			if distance.X() > max.X() {
				max = max.SetX(distance.X())
			}
			if distance.Y() > max.Y() {
				max = max.SetY(distance.Y())
			}
			if distance.Z() > max.Z() {
				max = max.SetZ(distance.Z())
			}

			if distance.X() < min.X() {
				min = min.SetX(distance.X())
			}
			if distance.Y() < min.Y() {
				min = min.SetY(distance.Y())
			}
			if distance.Z() < min.Z() {
				min = min.SetZ(distance.Z())
			}
		}

	}

	err = binary.Write(streamData, binary.LittleEndian, float32(maxTimeDifference))
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

	// Write starting position
	binary.Write(streamData, binary.LittleEndian, float32(captures[0].Position().X()))
	binary.Write(streamData, binary.LittleEndian, float32(captures[0].Position().Y()))
	binary.Write(streamData, binary.LittleEndian, float32(captures[0].Position().Z()))

	timeBuffer := make([]byte, 2)
	octBuffer := make([]OctCell, 8)
	octByteBuffer := make([]byte, 3)
	totalledQuantizedDuration := startingTime
	quantizedPosition := captures[0].Position()
	for i, capture := range captures {

		// Write Time
		duration := capture.Time() - totalledQuantizedDuration
		floatBSTToBytes(duration, 0, maxTimeDifference, timeBuffer)
		_, err := streamData.Write(timeBuffer)
		if err != nil {
			return nil, err
		}

		// Read back quantized time to fix drifting
		totalledQuantizedDuration += bytesToFloatBST(0, maxTimeDifference, timeBuffer)

		// Skip first since there will be no change in position from starting position
		if i > 0 {
			// Write position
			dir := capture.Position().Sub(quantizedPosition)
			Vec3ToOctCells(dir, min, max, octBuffer)
			octCellsToBytes24(octBuffer, octByteBuffer)
			_, err = streamData.Write(octByteBuffer)
			if err != nil {
				return nil, err
			}

			// Read back quantized position to fix drifting
			quantizedPosition = quantizedPosition.Add(OctCellsToVec3(min, max, octBuffer))
		}

	}

	return streamData.Bytes(), nil
}

func decodeOct24(streamData *bytes.Reader) ([]position.Capture, error) {
	numCaptures, err := binary.ReadUvarint(streamData)
	if err != nil {
		return nil, err
	}

	if numCaptures == 0 {
		return make([]position.Capture, 0), nil
	}

	var startTime float32
	err = binary.Read(streamData, binary.LittleEndian, &startTime)
	if err != nil {
		return nil, err
	}

	if numCaptures == 1 {
		var posX float32
		var posY float32
		var posZ float32
		err = binary.Read(streamData, binary.LittleEndian, &posX)
		err = binary.Read(streamData, binary.LittleEndian, &posY)
		err = binary.Read(streamData, binary.LittleEndian, &posZ)
		return []position.Capture{position.NewCapture(float64(startTime), float64(posX), float64(posY), float64(posZ))}, err
	}

	var maxTimeDifference float32
	err = binary.Read(streamData, binary.LittleEndian, &maxTimeDifference)
	if err != nil {
		return nil, err
	}

	var minX float32
	var minY float32
	var minZ float32
	var maxX float32
	var maxY float32
	var maxZ float32
	var startingX float32
	var startingY float32
	var startingZ float32

	err = binary.Read(streamData, binary.LittleEndian, &minX)
	err = binary.Read(streamData, binary.LittleEndian, &minY)
	err = binary.Read(streamData, binary.LittleEndian, &minZ)
	err = binary.Read(streamData, binary.LittleEndian, &maxX)
	err = binary.Read(streamData, binary.LittleEndian, &maxY)
	err = binary.Read(streamData, binary.LittleEndian, &maxZ)
	err = binary.Read(streamData, binary.LittleEndian, &startingX)
	err = binary.Read(streamData, binary.LittleEndian, &startingY)
	err = binary.Read(streamData, binary.LittleEndian, &startingZ)
	min := vector.NewVector3(float64(minX), float64(minY), float64(minZ))
	max := vector.NewVector3(float64(maxX), float64(maxY), float64(maxZ))
	starting := vector.NewVector3(float64(startingX), float64(startingY), float64(startingZ))

	captures := make([]position.Capture, numCaptures)
	timeBuffer := make([]byte, 2)
	octBuffer := make([]OctCell, 8)
	octBytesBuffer := make([]byte, 3)
	currentTime := float64(startTime)
	currentPosition := starting
	for i := 0; i < int(numCaptures); i++ {
		streamData.Read(timeBuffer)
		time := bytesToFloatBST(0, float64(maxTimeDifference), timeBuffer)
		currentTime += time

		if i > 0 {
			streamData.Read(octBytesBuffer)
			bytesToOctCells24(octBuffer, octBytesBuffer)
			v := OctCellsToVec3(min, max, octBuffer)
			currentPosition = currentPosition.Add(v)
		}

		captures[i] = position.NewCapture(currentTime, currentPosition.X(), currentPosition.Y(), currentPosition.Z())
	}

	return captures, nil
}