package options

import (
	"time"

	log "github.com/Sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Loglet struct {
	KinesisStream       string
	KinesisPartitionKey string
	KinesisRegion       string
	CursorFile          string
	MaxMessageDelay     time.Duration
	MaxMessageSize      int
	MaxMessageCount     int
	DefaultFields       map[string]string
	LogLevel            log.Level
	IncludeFilters      []string
	ExcludeFilters      []string

	// testing/debugging
	CpuProfile string
	MemProfile string
}

const (
	MB = 1 << (10 * 2)
)

func NewLoglet() *Loglet {
	return &Loglet{
		KinesisStream:       "k8s-logs",
		KinesisPartitionKey: "logs",
		KinesisRegion:       "eu-west-1",
		CursorFile:          "loglet.cursor",
		MaxMessageDelay:     10 * time.Second,
		MaxMessageSize:      1 * MB,
		MaxMessageCount:     2000,
		DefaultFields:       make(map[string]string),
		LogLevel:            log.InfoLevel,

		// testing/debugging
		CpuProfile: "",
		MemProfile: "",
	}
}

func (l *Loglet) AddFlags() {
	kingpin.Flag("stream", "kinesis stream to write to").Default(l.KinesisStream).StringVar(&l.KinesisStream)
	kingpin.Flag("partition-key", "kinesis partition key to use").Default(l.KinesisPartitionKey).StringVar(&l.KinesisPartitionKey)
	kingpin.Flag("region", "kinesis stream region").Default(l.KinesisRegion).StringVar(&l.KinesisRegion)
	kingpin.Flag("cursor-file", "File in which to keep cursor state between runs").Default(l.CursorFile).StringVar(&l.CursorFile)
	kingpin.Flag("default-field", "Default fields to add to all log entries. Values of fields in messages take precedence").StringMapVar(&l.DefaultFields)
	kingpin.Flag("log-level", "Log level").Default(l.LogLevel.String()).SetValue(&LogLevelValue{&l.LogLevel})
	kingpin.Flag("include-filter", "Include entries with a matching key value pair in the fields, combines as OR. Format: Key=Value").StringsVar(&l.IncludeFilters)
	kingpin.Flag("exclude-filter", "Exclude entries with a matching key value pair in the fields, combines as OR. Format: Key=Value").StringsVar(&l.ExcludeFilters)

	// TODO: might resurrect these if switching to AsyncProducer
	// kingpin.Flag("max-message-delay", "The maximum time to buffer messages before sending to ES.").Default(l.MaxMessageDelay.String()).DurationVar(&l.MaxMessageDelay)
	// kingpin.Flag("max-message-size", "The maximum size (soft limit) of a single message payload sent to ES.").Default(strconv.Itoa(l.MaxMessageSize)).IntVar(&l.MaxMessageSize)
	// kingpin.Flag("max-message-count", "The maximum number of messages to buffer before sending to ES.").Default(strconv.Itoa(l.MaxMessageCount)).IntVar(&l.MaxMessageCount)

	// hiden testing/debugging flags
	kingpin.Flag("cpu-profile", "").Hidden().Default(l.CpuProfile).StringVar(&l.CpuProfile)
	kingpin.Flag("mem-profile", "").Hidden().Default(l.MemProfile).StringVar(&l.MemProfile)
}

type LogLevelValue struct {
	target *log.Level
}

func (l *LogLevelValue) String() string {
	return l.target.String()
}

func (l *LogLevelValue) Set(s string) error {
	level, err := log.ParseLevel(s)
	if err != nil {
		return err
	}
	*l.target = level
	return nil
}
