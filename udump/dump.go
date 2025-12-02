package udump

import (
	"bytes"
	"encoding/json"
)

func Value(val any) []byte {
  jval, err := json.MarshalIndent(val, "", "  ")
  if err != nil {
    return []byte(err.Error())
  }
  return jval
}

func JSON(buf []byte) []byte {
  var jbuf bytes.Buffer
  err := json.Indent(&jbuf, buf, "", "  ")
  if err != nil {
    return []byte(err.Error())
  }
  return jbuf.Bytes()
}

func ValueRaw(val any) []byte {
  var jval bytes.Buffer
  enc := json.NewEncoder(&jval)
  enc.SetIndent("", "  ")
  enc.SetEscapeHTML(false)
  err := enc.Encode(val)
  if err != nil {
    return []byte(err.Error())
  }
  return jval.Bytes()
}

func JSONRaw(buf []byte) []byte {
  var val any
  err := json.Unmarshal(buf, &val)
  if err != nil {
    return []byte(err.Error())
  }
  return ValueRaw(val)
}
