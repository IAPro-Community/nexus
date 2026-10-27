package artifacts

import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "image"
 "image/color"
 "image/png"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func makeTestRoot(t *testing.T)(*os.Root,string){
 t.Helper()
 dir:=t.TempDir()
 for _,part:=range []string{"images",".git","node_modules","images/revisions"}{if err:=os.MkdirAll(filepath.Join(dir,part),0700);err!=nil{t.Fatal(err)}}
 imageBytes:=new(bytes.Buffer)
 img:=image.NewRGBA(image.Rect(0,0,8,8));img.Set(1,2,color.RGBA{R:230,G:60,B:90,A:255})
 if err:=png.Encode(imageBytes,img);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"images","a.png"),imageBytes.Bytes(),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"report.pdf"),[]byte("%PDF-1.4\nTest"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"notes.md"),[]byte("Generated output"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,".env"),[]byte("API_KEY=PRIVATE"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"credentials.json"),[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"images","fake.png"),[]byte("<script>alert('bad')</script>"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"node_modules","private.md"),[]byte("hidden"),0600);err!=nil{t.Fatal(err)}
 root,err:=Root(dir);if err!=nil{t.Fatal(err)}
 t.Cleanup(func(){_ =root.Close()})
 return root,dir
}
func TestProjectPathProtection(t *testing.T){
 root,dir:=makeTestRoot(t)
 for _,bad:=range []string{"../outside","/etc/passwd","images/../../.env",".env",".git/config","credentials.json","node_modules/private.md","images\\a.png","images//a.png","images/a.png/../.env",""}{
  if bad==""{continue}
  if _,err:=SafeRelative(bad);err==nil{t.Fatalf("accepted forbidden path %q",bad)}
 }
 outside:=filepath.Join(t.TempDir(),"outside.png")
 _=os.WriteFile(outside,[]byte("secret"),0600)
 if err:=os.Symlink(outside,filepath.Join(dir,"images","outside.png"));err==nil{
  if _,err:=Describe(root,"images/outside.png");err==nil{t.Fatal("escaped project root through symlink")}
 }
 if _,err:=Describe(root,"images/fake.png");err==nil{t.Fatal("HTML masquerading as PNG was served")}
}
func TestOutputScanAndFiles(t *testing.T){
 root,_:=makeTestRoot(t)
 list,err:=Scan(root,0,20,"");if err!=nil{t.Fatal(err)}
 if list.Total!=3{t.Fatalf("want 3 safe outputs, found %+v",list)}
 if list.Items[0].SHA256==""{t.Fatal("missing digest")}
 files,err:=Directory(root,".");if err!=nil{t.Fatal(err)}
 for _,item:=range files{
  if strings.HasPrefix(item.Name,".")||strings.Contains(item.Name,"credential")||item.Name=="node_modules"{t.Fatalf("unsafe entry visible: %+v",item)}
 }
 images,err:=Scan(root,0,1,"image")
 if err!=nil||images.Total!=1||len(images.Items)!=1||images.Items[0].Name!="a.png"{t.Fatalf("invalid images must be excluded: %+v %v",images,err)}
}
func TestServeAndThumbnail(t *testing.T){
 root,_:=makeTestRoot(t)
 r:=httptest.NewRequest(http.MethodGet,"/download",nil);w:=httptest.NewRecorder()
 if err:=Serve(w,r,root,"images/a.png",true);err!=nil{t.Fatal(err)}
 if got:=w.Code;got!=200{t.Fatalf("image: %d",got)}
 raw:=w.Body.Bytes();sum:=sha256.Sum256(raw)
 item,err:=Describe(root,"images/a.png");if err!=nil{t.Fatal(err)}
 if item.SHA256!=hex.EncodeToString(sum[:]){t.Fatal("download corrupted original")}
 if w.Header().Get("X-Content-Type-Options")!="nosniff"{t.Fatal("missing security headers")}
 thumb:=httptest.NewRecorder()
 if err:=Thumbnail(thumb,r,root,"images/a.png",t.TempDir(),"project-1");err!=nil{t.Fatal(err)}
 if thumb.Code!=200||thumb.Header().Get("Content-Type")!="image/jpeg"{t.Fatalf("thumbnail failed: %d %s",thumb.Code,thumb.Body.String())}
 if _,_,err:=image.Decode(bytes.NewReader(thumb.Body.Bytes()));err!=nil{t.Fatalf("thumbnail invalid: %v",err)}
 if err:=Thumbnail(httptest.NewRecorder(),r,root,"images/fake.png",t.TempDir(),"project-1");err==nil{t.Fatal("fake image generated thumbnail")}
}
