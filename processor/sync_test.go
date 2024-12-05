package processor

import (
	"github.com/qubic/go-archiver/protobuff"
	"testing"
	"time"
)

func TestSyncProcessor_CalculateSyncDelta(t *testing.T) {

	mockSyncProcessor := NewSyncProcessor(SyncConfiguration{}, nil, time.Second)

	testData := []struct {
		name                 string
		bootstrapMetadata    *protobuff.SyncMetadataResponse
		clientMetadata       *protobuff.SyncMetadataResponse
		lastSynchronizedTick *protobuff.SyncLastSynchronizedTick
		expected             SyncDelta
	}{
		{
			name: "TestCalculateSyncDelta_1",
			bootstrapMetadata: &protobuff.SyncMetadataResponse{
				ArchiverVersion:  "dev",
				MaxObjectRequest: 1000,
				ProcessedTickIntervals: []*protobuff.ProcessedTickIntervalsPerEpoch{
					{
						Epoch: 123,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15500000,
								LastProcessedTick:    15578849,
							},
						},
					},
					{
						Epoch: 124,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15590000,
								LastProcessedTick:    15694132,
							},
						},
					},
					{
						Epoch: 125,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15700000,
								LastProcessedTick:    15829438,
							},
						},
					},
					{
						Epoch: 126,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15840000,
								LastProcessedTick:    15959704,
							},
						},
					},
					{
						Epoch: 127,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15970000,
								LastProcessedTick:    16089394,
							},
						},
					},
				},
			},
			clientMetadata: &protobuff.SyncMetadataResponse{
				ArchiverVersion:  "dev",
				MaxObjectRequest: 1000,
				ProcessedTickIntervals: []*protobuff.ProcessedTickIntervalsPerEpoch{
					{
						Epoch: 124,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15590000,
								LastProcessedTick:    15694132,
							},
						},
					},
					{
						Epoch: 125,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15700000,
								LastProcessedTick:    15829438,
							},
						},
					},
					{
						Epoch: 126,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15840000,
								LastProcessedTick:    15849999,
							},
						},
					},
				},
			},
			lastSynchronizedTick: nil,
			expected: SyncDelta{
				{
					Epoch: 123,
					ProcessedIntervals: []*protobuff.ProcessedTickInterval{
						{
							InitialProcessedTick: 15500000,
							LastProcessedTick:    15578849,
						},
					},
				},
				{
					Epoch: 126,
					ProcessedIntervals: []*protobuff.ProcessedTickInterval{
						{
							InitialProcessedTick: 15840000,
							LastProcessedTick:    15959704,
						},
					},
				},
				{
					Epoch: 127,
					ProcessedIntervals: []*protobuff.ProcessedTickInterval{
						{
							InitialProcessedTick: 15970000,
							LastProcessedTick:    16089394,
						},
					},
				},
			},
		},
		{
			name: "TestCalculateSyncDelta_2",
			bootstrapMetadata: &protobuff.SyncMetadataResponse{
				ArchiverVersion:  "dev",
				MaxObjectRequest: 1000,
				ProcessedTickIntervals: []*protobuff.ProcessedTickIntervalsPerEpoch{
					{
						Epoch: 123,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15500000,
								LastProcessedTick:    15578849,
							},
						},
					},
					{
						Epoch: 124,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15590000,
								LastProcessedTick:    15694132,
							},
						},
					},
					{
						Epoch: 125,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15700000,
								LastProcessedTick:    15829438,
							},
						},
					},
					{
						Epoch: 126,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15840000,
								LastProcessedTick:    15959704,
							},
						},
					},
					{
						Epoch: 127,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15970000,
								LastProcessedTick:    16089394,
							},
						},
					},
				},
			},
			clientMetadata: &protobuff.SyncMetadataResponse{
				ArchiverVersion:  "dev",
				MaxObjectRequest: 1000,
				ProcessedTickIntervals: []*protobuff.ProcessedTickIntervalsPerEpoch{
					{
						Epoch: 124,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15590000,
								LastProcessedTick:    15694132,
							},
						},
					},
					{
						Epoch: 125,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15700000,
								LastProcessedTick:    15829438,
							},
						},
					},
					{
						Epoch: 126,
						Intervals: []*protobuff.ProcessedTickInterval{
							{
								InitialProcessedTick: 15840000,
								LastProcessedTick:    15849999,
							},
						},
					},
				},
			},
			lastSynchronizedTick: &protobuff.SyncLastSynchronizedTick{
				TickNumber: 15849999,
				Epoch:      126,
				ChainHash:  nil,
				StoreHash:  nil,
			},
			expected: SyncDelta{
				{
					Epoch: 123,
					ProcessedIntervals: []*protobuff.ProcessedTickInterval{
						{
							InitialProcessedTick: 15500000,
							LastProcessedTick:    15578849,
						},
					},
				},
				{
					Epoch: 126,
					ProcessedIntervals: []*protobuff.ProcessedTickInterval{
						{
							InitialProcessedTick: 15849999,
							LastProcessedTick:    15959704,
						},
					},
				},
				{
					Epoch: 127,
					ProcessedIntervals: []*protobuff.ProcessedTickInterval{
						{
							InitialProcessedTick: 15970000,
							LastProcessedTick:    16089394,
						},
					},
				},
			},
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			syncDelta, err := mockSyncProcessor.CalculateSyncDelta(data.bootstrapMetadata, data.clientMetadata, data.lastSynchronizedTick)
			if err != nil {
				t.Fatalf("Error occured while calculating sync data: %v", err)
			}

			if len(data.expected) != len(syncDelta) {
				t.Fatalf("Mismatched delta length. Expected: %d Got: %d", len(data.expected), len(syncDelta))
			}

			for index := 0; index < len(data.expected); index++ {
				expectedEpochDelta := data.expected[index]
				gotEpochDelta := syncDelta[index]
				if expectedEpochDelta.Epoch != gotEpochDelta.Epoch {
					t.Fatalf("Mismatched epoch at index %d. Expected: %d Got: %d", index, expectedEpochDelta.Epoch, gotEpochDelta.Epoch)
				}

				if len(expectedEpochDelta.ProcessedIntervals) != len(gotEpochDelta.ProcessedIntervals) {
					t.Fatalf("Mismatched processed interval list for epoch %d. Expected: %d Got: %d", expectedEpochDelta.Epoch, len(expectedEpochDelta.ProcessedIntervals), len(gotEpochDelta.ProcessedIntervals))
				}

				for intervalIndex := 0; intervalIndex < len(expectedEpochDelta.ProcessedIntervals); intervalIndex++ {

					expectedInterval := expectedEpochDelta.ProcessedIntervals[intervalIndex]
					gotInterval := gotEpochDelta.ProcessedIntervals[intervalIndex]

					if expectedInterval.InitialProcessedTick != gotInterval.InitialProcessedTick || expectedInterval.LastProcessedTick != gotInterval.LastProcessedTick {
						t.Fatalf("Mismatched tick intervals at index %d for epoch %d. \nExpected: %v \nGot: %v", intervalIndex, expectedEpochDelta.Epoch, expectedInterval, gotInterval)
					}

				}

			}
		})
	}

}
