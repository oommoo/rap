package io_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/recolude/rap/format"
	"github.com/recolude/rap/format/io"
	"github.com/recolude/rap/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_ErrorsWithNoValidEncoders(t *testing.T) {
	// ARRANGE ================================================================
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := mocks.NewMockCaptureStream(ctrl)
	stream.EXPECT().Signature().Return("test.data")

	rec := mocks.NewMockRecording(ctrl)
	rec.EXPECT().CaptureStreams().AnyTimes().Return([]format.CaptureStream{stream})

	w := io.NewWriter(nil, nil)

	// ACT ====================================================================
	_, err := w.Write(rec)

	// ASSERT =================================================================
	assert.EqualError(t, err, "no encoder registered to handle stream: test.data")
}

func Test_PanicsWithNilRecording(t *testing.T) {
	// ARRANGE ================================================================
	w := io.NewWriter(nil, nil)

	// ACT/ASSERT =============================================================
	assert.PanicsWithError(t, "can not write nil recording", func() { w.Write(nil) })
}