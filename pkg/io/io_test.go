package io_test

import (
	"bytes"
	"testing"

	"github.com/recolude/rap/pkg/data"
	"github.com/recolude/rap/pkg/encoding"
	positionEncoding "github.com/recolude/rap/pkg/encoding/position"
	"github.com/recolude/rap/pkg/io"
	"github.com/recolude/rap/pkg/streams/position"
	"github.com/stretchr/testify/assert"
)

func assertRecordingsMatch(t *testing.T, recExpected, recActual data.Recording) bool {
	if assert.Equal(t, len(recExpected.Binaries()), len(recActual.Binaries())) == false {
		return false
	}

	if assert.Equal(t, len(recExpected.Recordings()), len(recActual.Recordings()), "Mismatch child recordings") == false {
		return false
	}

	if assert.NotNil(t, recActual) == false {
		return false
	}

	if assert.Equal(t, recExpected.Name(), recActual.Name()) == false {
		return false
	}

	if assert.Equal(t, len(recExpected.Metadata()), len(recActual.Metadata())) == false {
		return false
	}

	for key, element := range recExpected.Metadata() {
		assert.Equal(t, element, recActual.Metadata()[key])
	}

	if assert.Equal(t, len(recExpected.CaptureStreams()), len(recActual.CaptureStreams())) == false {
		return false
	}

	for streamIndex, stream := range recExpected.CaptureStreams() {
		assert.Equal(t, stream.Name(), recActual.CaptureStreams()[streamIndex].Name())

		for i, correctCapture := range recExpected.CaptureStreams()[streamIndex].Captures() {
			assert.Equal(t, correctCapture.Time(), recActual.CaptureStreams()[streamIndex].Captures()[i].Time())
		}

	}

	return true
}

func Test_HandlesOneRecordingOneStream(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
	}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := data.NewRecording(
		"Test Recording",
		[]data.CaptureStream{
			position.NewStream(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
		},
		nil,
		map[string]string{
			"a":  "bee",
			"ce": "dee",
		},
		nil,
	)

	// ACT ====================================================================
	n, errWrite := w.Write(recIn)
	recOut, nOut, errRead := r.Read()

	// ASSERT =================================================================
	assert.NoError(t, errWrite)
	assert.NoError(t, errRead)
	assert.Equal(t, n, nOut)
	assertRecordingsMatch(t, recIn, recOut)
}

func Test_HandlesOneRecordingTwoStream(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
	}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := data.NewRecording(
		"Test Recording",
		[]data.CaptureStream{
			position.NewStream(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
			position.NewStream(
				"Position2",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
		},
		nil,
		map[string]string{
			"a":  "bee",
			"ce": "dee",
		},
		nil,
	)

	// ACT ====================================================================
	n, errWrite := w.Write(recIn)
	recOut, nOut, errRead := r.Read()

	// ASSERT =================================================================
	assert.NoError(t, errWrite)
	assert.NoError(t, errRead)
	assert.Equal(t, n, nOut)
	assertRecordingsMatch(t, recIn, recOut)
}

func Test_HandlesNestedRecordings(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
	}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := data.NewRecording(
		"Test Recording",
		[]data.CaptureStream{
			position.NewStream(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
			position.NewStream(
				"Position2",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
		},
		[]data.Recording{
			data.NewRecording(
				"Child Recording",
				[]data.CaptureStream{
					position.NewStream(
						"Child Position",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
					position.NewStream(
						"Child Position2",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
				},
				nil,
				map[string]string{
					"a":  "bee",
					"ce": "dee",
				},
				nil,
			),
		},
		map[string]string{
			"a":  "bee",
			"ce": "dee",
		},
		nil,
	)

	// ACT ====================================================================
	n, errWrite := w.Write(recIn)
	recOut, nOut, errRead := r.Read()

	// ASSERT =================================================================
	assert.NoError(t, errWrite)
	assert.NoError(t, errRead)
	assert.Equal(t, n, nOut)
	assertRecordingsMatch(t, recIn, recOut)
}
