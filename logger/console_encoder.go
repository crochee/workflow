package logger

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"io"
	"math"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	_pool = buffer.NewPool()
)

// For JSON-escaping; see consoleEncoder.safeAddString below.
const _hex = "0123456789abcdef"

var _jsonPool = sync.Pool{New: func() interface{} {
	return &consoleEncoder{}
}}

func getJSONEncoder() *consoleEncoder {
	return _jsonPool.Get().(*consoleEncoder)
}

func putJSONEncoder(enc *consoleEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_jsonPool.Put(enc)
}

var _sliceEncoderPool = sync.Pool{
	New: func() interface{} {
		return &sliceArrayEncoder{elems: make([]interface{}, 0, 2)}
	},
}

func getSliceEncoder() *sliceArrayEncoder {
	return _sliceEncoderPool.Get().(*sliceArrayEncoder)
}

func putSliceEncoder(e *sliceArrayEncoder) {
	e.elems = e.elems[:0]
	_sliceEncoderPool.Put(e)
}

type consoleEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	spaced         bool // include spaces after colons and commas
	openNamespaces int

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc zapcore.ReflectedEncoder
}

// NewJSONEncoder creates a fast, low-allocation JSON encoder. The encoder
// appropriately escapes all field keys and values.
//
// Note that the encoder doesn't deduplicate keys, so it's possible to produce
// a message like
//   {"foo":"bar","foo":"baz"}
// This is permitted by the JSON specification, but not encouraged. Many
// libraries will ignore duplicate key-value pairs (typically keeping the last
// pair) when unmarshaling, but users should attempt to avoid adding duplicate
// keys.
func NewConsoleEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return newConsoleEncoder(cfg, false)
}

func newConsoleEncoder(cfg zapcore.EncoderConfig, spaced bool) *consoleEncoder {
	if cfg.ConsoleSeparator == "" {
		// Use a default delimiter of '\t' for backwards compatibility
		cfg.ConsoleSeparator = "\t"
	}

	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}
	// If no EncoderConfig.NewReflectedEncoder is provided by the user, then use default
	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = func(w io.Writer) zapcore.ReflectedEncoder {
			enc := json.NewEncoder(w)
			// For consistency with our custom JSON encoder.
			enc.SetEscapeHTML(false)
			return enc
		}
	}

	return &consoleEncoder{
		EncoderConfig: &cfg,
		buf:           _pool.Get(),
		spaced:        spaced,
	}
}

func (enc *consoleEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *consoleEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *consoleEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *consoleEncoder) AddByteString(key string, val []byte) {
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *consoleEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *consoleEncoder) AddComplex128(key string, val complex128) {
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *consoleEncoder) AddComplex64(key string, val complex64) {
	enc.addKey(key)
	enc.AppendComplex64(val)
}

func (enc *consoleEncoder) AddDuration(key string, val time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *consoleEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *consoleEncoder) AddFloat32(key string, val float32) {
	enc.addKey(key)
	enc.AppendFloat32(val)
}

func (enc *consoleEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.AppendInt64(val)
}

func (enc *consoleEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = _pool.Get()
		enc.reflectEnc = enc.NewReflectedEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

var nullLiteralBytes = []byte("null")

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *consoleEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *consoleEncoder) AddReflected(key string, obj interface{}) error {
	valueBytes, err := enc.encodeReflected(obj)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *consoleEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

func (enc *consoleEncoder) AddString(key, val string) {
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *consoleEncoder) AddTime(key string, val time.Time) {
	enc.addKey(key)
	enc.AppendTime(val)
}

func (enc *consoleEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *consoleEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	enc.addElementSeparator()
	enc.buf.AppendByte('[')
	err := arr.MarshalLogArray(enc)
	enc.buf.AppendByte(']')
	return err
}

func (enc *consoleEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	// Close ONLY new openNamespaces that are created during
	// AppendObject().
	old := enc.openNamespaces
	enc.openNamespaces = 0
	enc.addElementSeparator()
	enc.buf.AppendByte('{')
	err := obj.MarshalLogObject(enc)
	enc.buf.AppendByte('}')
	enc.closeOpenNamespaces()
	enc.openNamespaces = old
	return err
}

func (enc *consoleEncoder) AppendBool(val bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *consoleEncoder) AppendByteString(val []byte) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	enc.buf.AppendByte('"')
}

// appendComplex appends the encoded form of the provided complex128 value.
// precision specifies the encoding precision for the real and imaginary
// components of the complex number.
func (enc *consoleEncoder) appendComplex(val complex128, precision int) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, precision)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		enc.buf.AppendByte('+')
	}
	enc.buf.AppendFloat(i, precision)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *consoleEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *consoleEncoder) AppendInt64(val int64) {
	enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *consoleEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *consoleEncoder) AppendString(val string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddString(val)
	enc.buf.AppendByte('"')
}

func (enc *consoleEncoder) AppendTimeLayout(time time.Time, layout string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.buf.AppendTime(time, layout)
	enc.buf.AppendByte('"')
}

func (enc *consoleEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *consoleEncoder) AppendUint64(val uint64) {
	enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *consoleEncoder) AddInt(k string, v int)         { enc.AddInt64(k, int64(v)) }
func (enc *consoleEncoder) AddInt32(k string, v int32)     { enc.AddInt64(k, int64(v)) }
func (enc *consoleEncoder) AddInt16(k string, v int16)     { enc.AddInt64(k, int64(v)) }
func (enc *consoleEncoder) AddInt8(k string, v int8)       { enc.AddInt64(k, int64(v)) }
func (enc *consoleEncoder) AddUint(k string, v uint)       { enc.AddUint64(k, uint64(v)) }
func (enc *consoleEncoder) AddUint32(k string, v uint32)   { enc.AddUint64(k, uint64(v)) }
func (enc *consoleEncoder) AddUint16(k string, v uint16)   { enc.AddUint64(k, uint64(v)) }
func (enc *consoleEncoder) AddUint8(k string, v uint8)     { enc.AddUint64(k, uint64(v)) }
func (enc *consoleEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }
func (enc *consoleEncoder) AppendComplex64(v complex64)    { enc.appendComplex(complex128(v), 32) }
func (enc *consoleEncoder) AppendComplex128(v complex128)  { enc.appendComplex(complex128(v), 64) }
func (enc *consoleEncoder) AppendFloat64(v float64)        { enc.appendFloat(v, 64) }
func (enc *consoleEncoder) AppendFloat32(v float32)        { enc.appendFloat(float64(v), 32) }
func (enc *consoleEncoder) AppendInt(v int)                { enc.AppendInt64(int64(v)) }
func (enc *consoleEncoder) AppendInt32(v int32)            { enc.AppendInt64(int64(v)) }
func (enc *consoleEncoder) AppendInt16(v int16)            { enc.AppendInt64(int64(v)) }
func (enc *consoleEncoder) AppendInt8(v int8)              { enc.AppendInt64(int64(v)) }
func (enc *consoleEncoder) AppendUint(v uint)              { enc.AppendUint64(uint64(v)) }
func (enc *consoleEncoder) AppendUint32(v uint32)          { enc.AppendUint64(uint64(v)) }
func (enc *consoleEncoder) AppendUint16(v uint16)          { enc.AppendUint64(uint64(v)) }
func (enc *consoleEncoder) AppendUint8(v uint8)            { enc.AppendUint64(uint64(v)) }
func (enc *consoleEncoder) AppendUintptr(v uintptr)        { enc.AppendUint64(uint64(v)) }

func (enc *consoleEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *consoleEncoder) clone() *consoleEncoder {
	clone := getJSONEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.buf = _pool.Get()
	return clone
}

func (enc *consoleEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := _pool.Get()

	// We don't want the entry's metadata to be quoted and escaped (if it's
	// encoded as strings), which means that we can't use the JSON encoder. The
	// simplest option is to use the memory encoder and fmt.Fprint.
	//
	// If this ever becomes a performance bottleneck, we can implement
	// ArrayEncoder for our plain-text format.
	arr := getSliceEncoder()
	if enc.TimeKey != "" && enc.EncodeTime != nil {
		enc.EncodeTime(ent.Time, arr)
	}
	if enc.LevelKey != "" && enc.EncodeLevel != nil {
		enc.EncodeLevel(ent.Level, arr)
	}
	if ent.LoggerName != "" && enc.NameKey != "" {
		nameEncoder := enc.EncodeName

		if nameEncoder == nil {
			// Fall back to FullNameEncoder for backward compatibility.
			nameEncoder = zapcore.FullNameEncoder
		}

		nameEncoder(ent.LoggerName, arr)
	}
	if ent.Caller.Defined {
		if enc.CallerKey != "" && enc.EncodeCaller != nil {
			enc.EncodeCaller(ent.Caller, arr)
		}
		if enc.FunctionKey != "" {
			arr.AppendString(ent.Caller.Function)
		}
	}
	for i := range arr.elems {
		if i > 0 {
			line.AppendString(enc.ConsoleSeparator)
		}
		fmt.Fprint(line, arr.elems[i])
	}
	putSliceEncoder(arr)

	// Add the message itself.
	if enc.MessageKey != "" {
		enc.addSeparatorIfNecessary(line)
		line.AppendString(ent.Message)
	}

	// Add any structured context.
	enc.writeContext(line, fields)

	// If there's no stacktrace key, honor that; this allows users to force
	// single-line output.
	if ent.Stack != "" && enc.StacktraceKey != "" {
		line.AppendByte('\n')
		line.AppendString(ent.Stack)
	}

	line.AppendString(enc.LineEnding)
	return line, nil
}

func (enc *consoleEncoder) truncate() {
	enc.buf.Reset()
}

func (enc *consoleEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
	enc.openNamespaces = 0
}

func (enc *consoleEncoder) addKey(key string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddString(key)
	enc.buf.AppendByte('"')
	enc.buf.AppendByte(':')
	if enc.spaced {
		enc.buf.AppendByte(' ')
	}
}

func (enc *consoleEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		enc.buf.AppendByte(',')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *consoleEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *consoleEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *consoleEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *consoleEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *consoleEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func (enc *consoleEncoder) addSeparatorIfNecessary(line *buffer.Buffer) {
	if line.Len() > 0 {
		line.AppendString(enc.ConsoleSeparator)
	}
}

func (enc *consoleEncoder) writeContext(line *buffer.Buffer, extra []zapcore.Field) {
	context := enc.clone()
	defer func() {
		// putJSONEncoder assumes the buffer is still used, but we write out the buffer so
		// we can free it.
		context.buf.Free()
		putJSONEncoder(context)
	}()

	addFields(context, extra)
	context.closeOpenNamespaces()
	if context.buf.Len() == 0 {
		return
	}

	enc.addSeparatorIfNecessary(line)
	line.AppendByte('{')
	line.Write(context.buf.Bytes())
	line.AppendByte('}')
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
