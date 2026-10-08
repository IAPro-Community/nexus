package protocol

import (
 "bytes"
 "encoding/json"
 "strings"
 "time"
)

// IsWireControlFrame matches the Nexus IPC envelope without treating arbitrary
// provider output containing "command" or "ok" as a control message.
func IsWireControlFrame(data []byte) bool {
 trimmed := bytes.TrimSpace(data)
 if len(trimmed) == 0 || trimmed[0] != '{' { return false }
 var fields map[string]json.RawMessage
 if json.Unmarshal(trimmed, &fields) != nil { return false }
 var version int
 if json.Unmarshal(fields["version"], &version) != nil || version != ProtocolVersion { return false }
 var stamp string
 if json.Unmarshal(fields["timestamp"], &stamp) != nil { return false }
 if _, err := time.Parse(time.RFC3339Nano, stamp); err != nil { return false }
 if raw, exists := fields["command"]; exists {
  var command, id string
  if json.Unmarshal(raw, &command) != nil || command == "" ||
   json.Unmarshal(fields["id"], &id) != nil || !strings.HasPrefix(id, "req-") { return false }
  suffix := strings.TrimPrefix(id, "req-")
  if suffix == "" { return false }
  for _, c := range suffix { if c < '0' || c > '9' { return false } }
  return true
 }
 var success bool
 return json.Unmarshal(fields["ok"], &success) == nil
}
