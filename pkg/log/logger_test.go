package log

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type fakeWriteSyncer struct {
	buf bytes.Buffer
}

type fields struct {
	Level   string
	Time    string
	Message string
	Name    string
	Age     int
}

func (fws *fakeWriteSyncer) Sync() error {
	return nil
}

func (fws *fakeWriteSyncer) Write(p []byte) (int, error) {
	return fws.buf.Write(p)
}

func (fws *fakeWriteSyncer) bytes() (p []byte) {
	s := fws.buf.Bytes()
	p = make([]byte, len(s))
	copy(p, s)
	fws.buf.Reset()
	return
}

func unmarshalLogMessage(t *testing.T, data []byte) *fields {
	var f fields
	err := json.Unmarshal(data, &f)
	assert.Nil(t, err, "failed to unmarshal log message: ", err)
	return &f
}

func TestLogger(t *testing.T) {
	fws := &fakeWriteSyncer{}
	logger, err := NewLogger(WithLogLevel("error"), WithWriteSyncer(fws))
	assert.Nil(t, err, "failed to new logger: ", err)
	defer logger.Close()

	logger.Error("hello")
	assert.Nil(t, logger.Sync(), "failed to sync logger")

	fields := unmarshalLogMessage(t, fws.bytes())
	assert.Equal(t, fields.Level, "error", "bad log level ", fields.Level)
	assert.Equal(t, fields.Message, "hello", "bad log message ", fields.Message)

	logger.Errorw("hello", zap.String("name", "alex"), zap.Int("age", 3))
	assert.Nil(t, logger.Sync(), "failed to sync logger")

	fields = unmarshalLogMessage(t, fws.bytes())
	assert.Equal(t, fields.Level, "error", "bad log level ", fields.Level)
	assert.Equal(t, fields.Message, "hello", "bad log message ", fields.Message)
	assert.Equal(t, fields.Name, "alex", "bad name field ", fields.Name)
	assert.Equal(t, fields.Age, 3, "bad age field ", fields.Age)

	logger.Warn("a non-visible message")
	assert.Nil(t, logger.Sync(), "failed to sync logger")

	data := fws.bytes()
	assert.Len(t, data, 0, "saw an unexpected log message")
}
