package options

import (
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Loglet struct {
	KafkaBrokers    []string
	KafkaTopic      string
	CursorFile      string
	MaxMessageDelay time.Duration
	MaxMessageSize  int
	MaxMessageCount int
	ExtraFields     map[string]string
	LogLevel        log.Level

	// testing/debugging
	FakeKafka  bool
	CpuProfile string
	MemProfile string
}

const (
	MB = 1 << (10 * 2)
)

func NewLoglet() *Loglet {
	return &Loglet{
		KafkaBrokers:    nil,
		KafkaTopic:      "logs",
		CursorFile:      "loglet.cursor",
		MaxMessageDelay: 10 * time.Second,
		MaxMessageSize:  1 * MB,
		MaxMessageCount: 2000,
		ExtraFields:     make(map[string]string),
		LogLevel:        log.InfoLevel,

		// testing/debugging
		FakeKafka:  false,
		CpuProfile: "",
		MemProfile: "",
	}
}

func (l *Loglet) AddFlags() {
	kingpin.Flag("broker", "kafka brokers in destination cluster").Default("localhost:9092").StringsVar(&l.KafkaBrokers)
	kingpin.Flag("topic", "kafka topic to produce messages to").Default(l.KafkaTopic).StringVar(&l.KafkaTopic)
	kingpin.Flag("cursor-file", "File in which to keep cursor state between runs").Default(l.CursorFile).StringVar(&l.CursorFile)
	kingpin.Flag("default-field", "Default extra fields to add to all log entries. Values of fields in messages take precedence").StringMapVar(&l.ExtraFields)
	kingpin.Flag("log-level", "Log level").Default(l.LogLevel.String()).SetValue(&LogLevelValue{&l.LogLevel})

	// TODO: might resurrect these if switching to AsyncProducer
	// kingpin.Flag("max-message-delay", "The maximum time to buffer messages before sending to ES.").Default(l.MaxMessageDelay.String()).DurationVar(&l.MaxMessageDelay)
	// kingpin.Flag("max-message-size", "The maximum size (soft limit) of a single message payload sent to ES.").Default(strconv.Itoa(l.MaxMessageSize)).IntVar(&l.MaxMessageSize)
	// kingpin.Flag("max-message-count", "The maximum number of messages to buffer before sending to ES.").Default(strconv.Itoa(l.MaxMessageCount)).IntVar(&l.MaxMessageCount)

	// hiden testing/debugging flags
	kingpin.Flag("fake-kafka", "").Hidden().Default(strconv.FormatBool(l.FakeKafka)).BoolVar(&l.FakeKafka)
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
