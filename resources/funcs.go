package resources

import (
	"fmt"
	"reflect"

	"github.com/rs/zerolog"
)

func toStrings(value any) []string {
	result := []string{}
	if reflect.TypeOf(value).Kind() == reflect.Slice {
		//logger.Debug().Msg("type is slice")
		slice := reflect.ValueOf(value)
		for i := 0; i < slice.Len(); i++ {
			switch v := slice.Index(i).Interface().(type) {
			case string:
				// logger.Debug().Msg("type is string")
				result = append(result, v)
			case []byte:
				// logger.Debug().Msg("type is []byte")
				result = append(result, string(v))
			case []string:
				// logger.Debug().Msg("type is []string")
				result = append(result, v...)
			default:
				result = append(result, fmt.Sprintf("%v", v))
				// logger.Debug().Str("type", fmt.Sprintf("%T", v)).Msg("type of data")
			}
		}
	}
	if len(result) > 0 {
		return result
	}
	return nil
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int8, int16, int32, int64, uint16, uint32, uint64, byte:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func makeLog(logger zerolog.Logger) func(msg string, args ...any) {
	return func(msg string, args ...any) {
		logger.Info().Msg(fmt.Sprintf(msg, args...))
	}
}
