package kinesis

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/publisher"
)

func init() {
	outputs.RegisterType("kinesis", makeKinesis)
}

type kinesisOutput struct {
	beat     beat.Info
	observer outputs.Observer
	codec    codec.Codec
	stream   string
	region   string
}

// New instantiates a new kinesis output instance.
func makeKinesis(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	// disable bulk support in publisher pipeline
	cfg.SetInt("bulk_max_size", -1, -1)

	ko := &kinesisOutput{beat: beat, observer: observer}
	if err := ko.init(beat, config); err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(-1, 0, ko)
}

func (out *kinesisOutput) init(beat beat.Info, config config) error {
	var err error
	out.stream = config.Stream
	out.region = config.Region

	enc, err := codec.CreateEncoder(beat, config.Codec)
	if err != nil {
		return err
	}

	out.codec = enc

    logp.Info("Using stream %v in region %v", out.stream, out.region)

	return nil
}

// Implement Outputer
func (out *kinesisOutput) Close() error {
	return nil
}

func (out *kinesisOutput) Publish(
	batch publisher.Batch,
) error {
	defer batch.ACK()

	st := out.observer
	events := batch.Events()
	st.NewBatch(len(events))

	dropped := 0

	for i := range events {
		event := &events[i]
		serializedEvent, err := out.codec.Encode(out.beat.Beat, &event.Content)

		awsSession := session.New(&aws.Config{Region: aws.String(out.region)})
		kc := kinesis.New(awsSession)

		streamName := aws.String(out.stream)
		streams, err := kc.DescribeStream(&kinesis.DescribeStreamInput{StreamName: streamName})
        logp.Info("Available Streams: %v\n", streams)

		entries := make([]*kinesis.PutRecordsRequestEntry, 1)
		for i := 0; i < len(entries); i++ {
			entries[i] = &kinesis.PutRecordsRequestEntry{
				Data:         []byte(serializedEvent),
				PartitionKey: aws.String("key2"),
			}
		}
		putsOutput, err := kc.PutRecords(&kinesis.PutRecordsInput{
			Records:    entries,
			StreamName: streamName,
		})
		if err != nil {
			panic(err)
		}
		// putsOutput has Records, and its shard id and sequece enumber.
		logp.Info("%v\n", putsOutput)

		if err != nil {
			if event.Guaranteed() {
				logp.Critical("Failed to serialize the event: %v", err)
			} else {
				logp.Warn("Failed to serialize the event: %v", err)
			}

			dropped++
			continue
		}

		st.WriteBytes(len(serializedEvent) + 1)
	}

	st.Dropped(dropped)
	st.Acked(len(events) - dropped)

	return nil
}
