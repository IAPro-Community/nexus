// Package agentwire contains shared process-stream protocols for agent harnesses.
// It does not own provider authentication, permission policies or orchestration.
package agentwire

import (
 "bufio"
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "sync"
)
const MaxFrame=4<<20
var ErrFrameTooLarge=errors.New("agentwire: frame exceeds maximum size")
var ErrClosed=errors.New("agentwire: connection closed")

// JSONLines is LF framed, unlike a Unicode line scanner. U+2028 and U+2029
// inside JSON string literals must be preserved, not mistaken for delimiters.
type JSONLines struct {
 reader *bufio.Reader
 writer io.Writer
 mu sync.Mutex
 closed bool
}
func NewJSONLines(input io.Reader,output io.Writer)*JSONLines{
 return &JSONLines{reader:bufio.NewReader(input),writer:output}
}
func(j *JSONLines)Read()(json.RawMessage,error){
 var data []byte
 for {
  fragment,err:=j.reader.ReadSlice('\n')
  if len(data)+len(fragment)>MaxFrame{return nil,ErrFrameTooLarge}
  data=append(data,fragment...)
  if err==bufio.ErrBufferFull{continue}
  if err!=nil{
   if err==io.EOF && len(data)==0{return nil,io.EOF}
   return nil,fmt.Errorf("agentwire: incomplete record: %w",err)
  }
  data=bytes.TrimSuffix(data,[]byte{'\n'})
  data=bytes.TrimSuffix(data,[]byte{'\r'})
  if !json.Valid(data)||len(data)==0{return nil,errors.New("agentwire: invalid JSON line")}
  return json.RawMessage(data),nil
 }
}
func(j *JSONLines)Write(value any)error{
 data,err:=json.Marshal(value)
 if err!=nil{return err}
 if len(data)>MaxFrame{return ErrFrameTooLarge}
 j.mu.Lock();defer j.mu.Unlock()
 if j.closed{return ErrClosed}
 frame:=append(data,'\n')
 n,err:=j.writer.Write(frame)
 if err!=nil{return err}
 if n!=len(frame){return io.ErrShortWrite}
 return nil
}
func(j *JSONLines)Close(){j.mu.Lock();defer j.mu.Unlock();j.closed=true}
