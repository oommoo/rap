package io_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/recolude/rap/format"
	"github.com/recolude/rap/format/collection/enum"
	"github.com/recolude/rap/format/collection/euler"
	"github.com/recolude/rap/format/collection/event"
	"github.com/recolude/rap/format/collection/position"
	"github.com/recolude/rap/format/encoding"
	enumEncoding "github.com/recolude/rap/format/encoding/enum"
	eulerEncoding "github.com/recolude/rap/format/encoding/euler"
	eventEncoding "github.com/recolude/rap/format/encoding/event"
	positionEncoding "github.com/recolude/rap/format/encoding/position"
	"github.com/recolude/rap/format/io"
	"github.com/recolude/rap/format/metadata"
	"github.com/recolude/rap/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func assertRecordingsMatch(t *testing.T, recExpected, recActual format.Recording) bool {
	if recExpected != nil && recActual == nil {
		t.Error("Expected recording to not be nil, but it was")
		return false
	}

	if assert.Equal(t, len(recExpected.ID()), len(recActual.ID())) == false {
		return false
	}

	if assert.Equal(t, len(recExpected.Binaries()), len(recActual.Binaries())) == false {
		return false
	}

	if assert.Equal(t, len(recExpected.BinaryReferences()), len(recActual.BinaryReferences())) == false {
		return false
	}

	for i, actual := range recActual.BinaryReferences() {
		assert.Equal(t, recExpected.BinaryReferences()[i].Name(), actual.Name())
		assert.Equal(t, recExpected.BinaryReferences()[i].URI(), actual.URI())
		assert.Equal(t, recExpected.BinaryReferences()[i].Size(), actual.Size())

		if assert.Equal(t, len(recExpected.BinaryReferences()[i].Metadata().Mapping()), len(actual.Metadata().Mapping()), "mismatch binary reference metadata lengths") == false {
			return false
		}

		for key, element := range recExpected.BinaryReferences()[i].Metadata().Mapping() {
			assert.Equal(t, element, recActual.BinaryReferences()[i].Metadata().Mapping()[key])
		}
	}

	if assert.Equal(t, len(recExpected.Recordings()), len(recActual.Recordings()), "Mismatch child recordings") == false {
		return false
	}

	for i, childRec := range recActual.Recordings() {
		if assertRecordingsMatch(t, recExpected.Recordings()[i], childRec) == false {
			return false
		}
	}

	if assert.NotNil(t, recActual) == false {
		return false
	}

	if assert.Equal(t, recExpected.Name(), recActual.Name()) == false {
		return false
	}

	if assert.Equal(t, len(recExpected.Metadata().Mapping()), len(recActual.Metadata().Mapping())) == false {
		return false
	}

	for key, element := range recExpected.Metadata().Mapping() {
		assert.Equal(t, element, recActual.Metadata().Mapping()[key])
	}

	if assert.Equal(t, len(recExpected.CaptureCollections()), len(recActual.CaptureCollections())) == false {
		return false
	}

	for streamIndex, stream := range recExpected.CaptureCollections() {
		assert.Equal(t, stream.Name(), recActual.CaptureCollections()[streamIndex].Name())

		for i, correctCapture := range recExpected.CaptureCollections()[streamIndex].Captures() {
			assert.Equal(t, correctCapture.Time(), recActual.CaptureCollections()[streamIndex].Captures()[i].Time())
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

	recIn := format.NewRecording(
		"44",
		"Test Recording",
		[]format.CaptureCollection{
			position.NewCollection(
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
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
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

func Test_HandlesBinaryReference(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	binaryRef := mocks.NewMockBinaryReference(ctrl)
	binaryRef.EXPECT().Name().AnyTimes().Return("Test Bin Ref")
	binaryRef.EXPECT().URI().AnyTimes().Return("recolude://orgid.projectid/binary-asset")
	binaryRef.EXPECT().Size().AnyTimes().Return(uint64(1234))
	binaryRef.EXPECT().Metadata().AnyTimes().Return(metadata.NewBlock(map[string]metadata.Property{
		"a": metadata.NewStringProperty("b"),
	}))

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := format.NewRecording(
		"44",
		"Test Recording",
		[]format.CaptureCollection{},
		nil,
		metadata.NewBlock(
			map[string]metadata.Property{
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
		[]format.BinaryReference{binaryRef},
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

	recIn := format.NewRecording(
		"",
		"Test Recording",
		[]format.CaptureCollection{
			position.NewCollection(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
			position.NewCollection(
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
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
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

	recIn := format.NewRecording(
		"",
		"Test Recording",
		[]format.CaptureCollection{
			position.NewCollection(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
			position.NewCollection(
				"Position2",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
		},
		[]format.Recording{
			format.NewRecording(
				"",
				"Child Recording",
				[]format.CaptureCollection{
					position.NewCollection(
						"Child Position",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
					position.NewCollection(
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
				metadata.NewBlock(
					map[string]metadata.Property{
						"a":  metadata.NewStringProperty("bee"),
						"ce": metadata.NewStringProperty("dee"),
					},
				),
				nil,
				nil,
			),
		},
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
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

func Test_EncodersWithHeaders(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
		eulerEncoding.NewEncoder(eulerEncoding.Raw64),
		eventEncoding.NewEncoder(eventEncoding.Raw64),
		enumEncoding.NewEncoder(enumEncoding.Raw32),
	}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := format.NewRecording(
		"",
		"Test Recording",
		[]format.CaptureCollection{
			position.NewCollection(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
				},
			),
		},
		[]format.Recording{
			format.NewRecording(
				"",
				"Child",
				[]format.CaptureCollection{
					event.NewCollection("ahhh", []event.Capture{
						event.NewCapture(
							1,
							"att",
							metadata.NewBlock(map[string]metadata.Property{
								"1": metadata.NewStringProperty("2"),
							})),
					}),
				},
				nil,
				metadata.EmptyBlock(),
				nil,
				nil,
			),
		},
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
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

func Test_HandlesMultipleEncoders(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
		eulerEncoding.NewEncoder(eulerEncoding.Raw64),
		eventEncoding.NewEncoder(eventEncoding.Raw64),
		enumEncoding.NewEncoder(enumEncoding.Raw32),
	}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := format.NewRecording(
		"",
		"Test Recording",
		[]format.CaptureCollection{
			position.NewCollection(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
			position.NewCollection(
				"Position2",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
		},
		[]format.Recording{
			format.NewRecording(
				"",
				"Child Recording",
				[]format.CaptureCollection{
					event.NewCollection("ahhh", []event.Capture{
						event.NewCapture(1, "att", metadata.NewBlock(map[string]metadata.Property{
							"1": metadata.NewStringProperty("2"),
						})),
					}),
					position.NewCollection(
						"Child Position",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
					position.NewCollection(
						"Child Position2",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
					euler.NewCollection(
						"Rot",
						[]euler.Capture{
							euler.NewEulerZXYCapture(1, 1, 2, 3),
							euler.NewEulerZXYCapture(2, 4, 5, 6),
							euler.NewEulerZXYCapture(4, 7, 8, 9),
							euler.NewEulerZXYCapture(7, 10, 11, 12),
						},
					),
					enum.NewCollection(
						"cmon",
						[]string{"A", "n"},
						[]enum.Capture{
							enum.NewCapture(1, 1),
						},
					),
				},
				nil,
				metadata.NewBlock(
					map[string]metadata.Property{
						"a":  metadata.NewStringProperty("bee"),
						"ce": metadata.NewStringProperty("dee"),
					},
				),
				nil,
				nil,
			),
			format.NewRecording(
				"",
				"Child 2 Recording",
				[]format.CaptureCollection{
					position.NewCollection(
						"Child Position",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
					position.NewCollection(
						"Child Position2",
						[]position.Capture{
							position.NewCapture(1, 1, 2, 3),
							position.NewCapture(2, 4, 5, 6),
							position.NewCapture(4, 7, 8, 9),
							position.NewCapture(7, 10, 11, 12),
						},
					),
					euler.NewCollection(
						"Rot",
						[]euler.Capture{
							euler.NewEulerZXYCapture(1, 1, 2, 3),
							euler.NewEulerZXYCapture(2, 4, 5, 6),
							euler.NewEulerZXYCapture(4, 7, 8, 9),
							euler.NewEulerZXYCapture(7, 10, 11, 12),
						},
					),
				},
				nil,
				metadata.NewBlock(
					map[string]metadata.Property{
						"a":  metadata.NewStringProperty("bee"),
						"ce": metadata.NewStringProperty("dee"),
					},
				),
				nil,
				nil,
			),
		},
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
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

func Test_HandlesManyChildren(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
		eulerEncoding.NewEncoder(eulerEncoding.Raw64),
		eventEncoding.NewEncoder(eventEncoding.Raw64),
		enumEncoding.NewEncoder(enumEncoding.Raw32),
	}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	childRec := format.NewRecording(
		"",
		"Child Recording",
		[]format.CaptureCollection{
			event.NewCollection("ahhh", []event.Capture{
				event.NewCapture(1, "att", metadata.NewBlock(map[string]metadata.Property{
					"1": metadata.NewStringProperty("2"),
				})),
			}),
			position.NewCollection(
				"Child Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
			euler.NewCollection(
				"Rot",
				[]euler.Capture{
					euler.NewEulerZXYCapture(1, 1, 2, 3),
					euler.NewEulerZXYCapture(2, 4, 5, 6),
					euler.NewEulerZXYCapture(4, 7, 8, 9),
					euler.NewEulerZXYCapture(7, 10, 11, 12),
				},
			),
			enum.NewCollection(
				"cmon",
				[]string{"A", "n"},
				[]enum.Capture{
					enum.NewCapture(1, 1),
				},
			),
		},
		nil,
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
		nil,
	)

	numChildren := 1600
	chilren := make([]format.Recording, numChildren)
	for i := range chilren {
		chilren[i] = childRec
	}

	recIn := format.NewRecording(
		"",
		"Test Recording",
		[]format.CaptureCollection{
			position.NewCollection(
				"Position",
				[]position.Capture{
					position.NewCapture(1, 1, 2, 3),
					position.NewCapture(2, 4, 5, 6),
					position.NewCapture(4, 7, 8, 9),
					position.NewCapture(7, 10, 11, 12),
				},
			),
		},
		chilren,
		metadata.NewBlock(
			map[string]metadata.Property{
				"a":  metadata.NewStringProperty("bee"),
				"ce": metadata.NewStringProperty("dee"),
			},
		),
		nil,
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

func Test_Uprade(t *testing.T) {
	f, err := os.Open(filepath.Join(v1DirectoryTestData, "Demo 38subj v1.rap"))
	if assert.NoError(t, err) == false {
		return
	}

	allBytes, err := ioutil.ReadAll(f)
	if assert.NoError(t, err) == false {
		return
	}

	encoders := []encoding.Encoder{
		positionEncoding.NewEncoder(positionEncoding.Raw64),
		eulerEncoding.NewEncoder(eulerEncoding.Raw64),
		eventEncoding.NewEncoder(eventEncoding.Raw64),
		enumEncoding.NewEncoder(enumEncoding.Raw32),
	}
	fileData := new(bytes.Buffer)

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	// ACT ====================================================================
	rec, _, err := io.Load(bytes.NewReader(allBytes))
	n, errWrite := w.Write(rec)
	recOut, nOut, errRead := r.Read()

	// ASSERT =================================================================
	assert.NoError(t, errWrite)
	assert.NoError(t, errRead)
	assert.Equal(t, n, nOut)
	assertRecordingsMatch(t, rec, recOut)
}

func Test_Metadata(t *testing.T) {
	// ARRANGE ================================================================
	fileData := new(bytes.Buffer)

	encoders := []encoding.Encoder{}

	w := io.NewWriter(encoders, fileData)
	r := io.NewReader(encoders, fileData)

	recIn := format.NewRecording(
		"",
		"Test MetadataRecording",
		[]format.CaptureCollection{},
		[]format.Recording{},
		metadata.NewBlock(
			map[string]metadata.Property{
				"int prop 77":     metadata.NewIntProperty(77),
				"int prop -100":   metadata.NewIntProperty(100),
				"int prop -0":     metadata.NewIntProperty(0),
				"string prop":     metadata.NewStringProperty("dee"),
				"bool true":       metadata.NewBoolProperty(true),
				"bool false":      metadata.NewBoolProperty(false),
				"byte test":       metadata.NewByteProperty(22),
				"vec2 test":       metadata.NewVector2Property(1.2, 3.4),
				"vec3 test":       metadata.NewVector3Property(1.2, 3.4, 5.6),
				"quaternion test": metadata.NewQuaternionProperty(1.2, 3.4, 5.6, 7.8),
				"matrix3x3 test":  metadata.NewMatrix3x3Property(1.2, 3.4, 5.6, 7.8, 9.10, 11.12, 13.14, 15.16, 17.18),
				"matrix4x4 test":  metadata.NewMatrix4x4Property(1.2, 3.4, 5.6, 7.8, 9.10, 11.12, 13.14, 15.16, 17.18, 19.20, 21.22, 23.24, 25.26, 27.28, 29.30, 31.32),
				"matrix test": metadata.NewMetadataProperty(metadata.NewBlock(
					map[string]metadata.Property{
						"nested prop 1": metadata.NewStringProperty("God kill me"),
						"nested prop 2": metadata.NewStringProperty("ahhhh"),
						"nested prop 3": metadata.NewIntProperty(666),
					},
				)),
			},
		),
		nil,
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
