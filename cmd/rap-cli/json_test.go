package main

import (
	"bytes"
	"testing"

	"github.com/recolude/rap/format"
	"github.com/recolude/rap/format/collection/position"
	"github.com/recolude/rap/format/io"
	"github.com/recolude/rap/format/metadata"
	"github.com/stretchr/testify/assert"
)

func Test_JSON(t *testing.T) {
	// ARRANGE ================================================================
	appIn := bytes.Buffer{}
	appOut := bytes.Buffer{}
	appErrOut := bytes.Buffer{}
	app := BuildApp(&appIn, &appOut, &appErrOut)
	if assert.NotNil(t, app) == false {
		return
	}

	rapWriter := io.NewRecoludeWriter(&appIn)
	_, writeErr := rapWriter.Write(
		format.NewRecording(
			"",
			"parent",
			[]format.CaptureCollection{
				position.NewCollection(
					"Position",
					[]position.Capture{
						position.NewCapture(1, 2, 3, 4),
					},
				),
			},
			[]format.Recording{
				format.NewRecording("sub-id", "sub-name", nil, nil, metadata.EmptyBlock(), nil, nil),
				format.NewRecording(
					"sub-id2",
					"sub-name2",
					[]format.CaptureCollection{
						position.NewCollection(
							"Position",
							[]position.Capture{
								position.NewCapture(1, 2, 3, 4),
							},
						),
						position.NewCollection(
							"Position2",
							[]position.Capture{
								position.NewCapture(1, 2, 3, 4),
							},
						),
					},
					nil,
					metadata.EmptyBlock(),
					nil,
					nil,
				),
			},
			metadata.NewBlock(map[string]metadata.Property{
				"xyz": metadata.NewBoolProperty(true),
			}),
			nil,
			nil,
		),
	)

	// ACT ====================================================================
	err := app.Run([]string{"rap-cli", "json"})

	// ASSERT =================================================================
	assert.NoError(t, err)
	assert.NoError(t, writeErr)
	assert.Equal(t, "", string(appErrOut.Bytes()))
	assert.Equal(t, `{
	"id": "",
	"name": "parent",
	"metadata": {"xyz":true},
	"collections": [
		{
			"name": "Position",
			"signature" : "recolude.position"
		}
	],
	"recordings": [
		{
			"id": "sub-id",
			"name": "sub-name",
			"metadata": {},
			"collections": [],
			"recordings": []
		},
		{
			"id": "sub-id2",
			"name": "sub-name2",
			"metadata": {},
			"collections": [
				{
					"name": "Position",
					"signature" : "recolude.position"
				},
				{
					"name": "Position2",
					"signature" : "recolude.position"
				}
			],
			"recordings": []
		}
	]
}`, string(appOut.Bytes()))
}
