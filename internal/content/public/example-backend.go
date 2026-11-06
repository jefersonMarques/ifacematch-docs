// example-backend.go — Customer Hosted standalone liveness, enrollment and 1:1 example.
// Run only on a developer workstation with Go 1.22+; never expose :8099 publicly.
// The application API key is kept in a private file, never served to the browser.
package main

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "log"
 "net/http"
 "net/url"
 "os"
 "regexp"
 "strings"
 "time"
)

var subjectPattern=regexp.MustCompile("^[A-Za-z0-9_-]{3,64}$")
var tokenPattern=regexp.MustCompile("^identity=[A-Za-z0-9_-]{32,128}$")
var checkIDPattern=regexp.MustCompile("^chk_[A-Za-z0-9_-]{16}$")

func sameOriginExample(r *http.Request)bool{
 origin:=r.Header.Get("Origin")
 return origin=="http://127.0.0.1:8099" || origin=="http://localhost:8099"
}

func issuerClient()*http.Client{
 return &http.Client{
  Timeout:15*time.Second,
  CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse},
 }
}


func privateAPIKey(path string)(string,error){
 if path=="" {return "",errors.New("IFACEMATCH_EXAMPLE_API_KEY_FILE is required")}
 info,err:=os.Lstat(path)
 if err!=nil||!info.Mode().IsRegular()||info.Mode().Perm()&0077!=0||info.Size()>1024 {
  return "",errors.New("API key must be in a private regular file (0600)")
 }
 raw,err:=os.ReadFile(path)
 if err!=nil{return "",err}
 defer clear(raw)
 key:=strings.TrimSpace(string(raw))
 if len(key)<32||len(key)>512 {return "",errors.New("invalid API key")}
 return key,nil
}
func validatedAPIOrigin(value string)(*url.URL,error){
 endpoint,err:=url.Parse(value)
 if err!=nil||endpoint.Scheme!="https"||endpoint.Host==""||
  endpoint.User!=nil||endpoint.Path!=""||endpoint.RawQuery!=""||endpoint.Fragment!=""{
  return nil,errors.New("IFACEMATCH_EXAMPLE_API_ORIGIN must be an exact HTTPS origin")
 }
 return endpoint,nil
}

// issuerPayload selects only the three existing commercial Server routes.
// The standalone liveness product does not require or send a subject ID.
func issuerPayload(flow,subjectID string)(string,[]byte,error){
 switch flow {
 case "liveness":
  return "/v1/liveness/checks",[]byte("{}"),nil
 case "enroll","verify":
  if !subjectPattern.MatchString(subjectID){return "",nil,errors.New("invalid subject ID")}
  endpoint:="/v1/enrollments"
  if flow=="verify"{endpoint="/v1/verifications"}
  body,err:=json.Marshal(map[string]string{"subjectId":subjectID})
  return endpoint,body,err
 default:
  return "",nil,errors.New("unsupported biometric flow")
 }
}

func startHandler(apiOrigin *url.URL, key string)http.HandlerFunc{
 client:=issuerClient()
 return func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("Content-Type","application/json; charset=utf-8")
  if r.Method!=http.MethodPost{
   http.Error(w,"POST required",http.StatusMethodNotAllowed);return
  }
  if !sameOriginExample(r) {
   http.Error(w,"invalid origin",http.StatusForbidden);return
  }
  r.Body=http.MaxBytesReader(w,r.Body,2048)
  defer r.Body.Close()
  var body struct{
   Flow string `json:"flow"`
   SubjectID string `json:"subjectId"`
  }
  decoder:=json.NewDecoder(r.Body)
  decoder.DisallowUnknownFields()
  if decoder.Decode(&body)!=nil {
   http.Error(w,"invalid body",http.StatusBadRequest);return
  }
  var extra any
  if !errors.Is(decoder.Decode(&extra),io.EOF){
   http.Error(w,"invalid JSON",http.StatusBadRequest);return
  }
  endpoint,encoded,err:=issuerPayload(body.Flow,body.SubjectID)
  if err!=nil{http.Error(w,"invalid biometric operation",http.StatusBadRequest);return}
  req,err:=http.NewRequestWithContext(r.Context(),http.MethodPost,
   apiOrigin.String()+endpoint,bytes.NewReader(encoded))
  if err!=nil{http.Error(w,"invalid request",500);return}
  req.Header.Set("Authorization","Bearer "+key)
  req.Header.Set("Content-Type","application/json")
  resp,err:=client.Do(req)
  if err!=nil {http.Error(w,"iFaceMatch Server unavailable",http.StatusBadGateway);return}
  defer resp.Body.Close()
  if resp.StatusCode!=http.StatusCreated{
   http.Error(w,"Server denied check creation (check scope, license or enrollment)",http.StatusBadGateway);return
  }
  var result struct{
   ID string `json:"id"`
   VerificationURL string `json:"verificationUrl"`
  }
  if json.NewDecoder(io.LimitReader(resp.Body,4096)).Decode(&result)!=nil{
   http.Error(w,"invalid Server response",http.StatusBadGateway);return
  }
  verificationURL,err:=url.Parse(result.VerificationURL)
  if err!=nil||!checkIDPattern.MatchString(result.ID)||
   verificationURL.Scheme!="https"||verificationURL.Host!=apiOrigin.Host||
   verificationURL.Path!="/verify.html"||!tokenPattern.MatchString(verificationURL.Fragment){
   http.Error(w,"unexpected verification URL",http.StatusBadGateway);return
  }
  _=json.NewEncoder(w).Encode(map[string]string{
   "id":result.ID,"verificationUrl":result.VerificationURL,
  })
 }
}

// statusHandler reads authoritative face decisions through the customer's
// application credential. It never sends the issuer secret to the browser.
func statusHandler(apiOrigin *url.URL,key string,client *http.Client)http.HandlerFunc{
 return func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("Content-Type","application/json; charset=utf-8")
  if r.Method!=http.MethodPost{
   http.Error(w,"POST required",http.StatusMethodNotAllowed);return
  }
  if !sameOriginExample(r){
   http.Error(w,"invalid origin",http.StatusForbidden);return
  }
  r.Body=http.MaxBytesReader(w,r.Body,1024)
  defer r.Body.Close()
  var body struct{
   Flow string `json:"flow"`
   CheckID string `json:"checkId"`
  }
  decoder:=json.NewDecoder(r.Body)
  decoder.DisallowUnknownFields()
  if err:=decoder.Decode(&body);err!=nil||!checkIDPattern.MatchString(body.CheckID){
   http.Error(w,"invalid check",http.StatusBadRequest);return
  }
  var extra any
  if !errors.Is(decoder.Decode(&extra),io.EOF){
   http.Error(w,"invalid JSON",http.StatusBadRequest);return
  }
  route:=""
  switch body.Flow{
  case "liveness":route="/v1/liveness/checks/"
  case "enroll":route="/v1/enrollments/"
  case "verify":route="/v1/verifications/"
  default:http.Error(w,"unsupported flow",http.StatusBadRequest);return
  }
  request,err:=http.NewRequestWithContext(r.Context(),http.MethodGet,
   apiOrigin.String()+route+body.CheckID,nil)
  if err!=nil{http.Error(w,"invalid request",http.StatusInternalServerError);return}
  request.Header.Set("Authorization","Bearer "+key)
  request.Header.Set("Accept","application/json")
  response,err:=client.Do(request)
  if err!=nil{http.Error(w,"iFaceMatch Server unavailable",http.StatusBadGateway);return}
  defer response.Body.Close()
  if response.StatusCode!=http.StatusOK{
   http.Error(w,"Server denied check status",http.StatusBadGateway);return
  }
  var result struct{
   ID string `json:"id"`
   Status string `json:"status"`
   Decision string `json:"decision"`
   Similarity float64 `json:"similarity"`
   LivenessScore float64 `json:"livenessScore"`
  }
  if err:=json.NewDecoder(io.LimitReader(response.Body,4096)).Decode(&result);err!=nil||
   result.ID!=body.CheckID||result.Status=="" {
   http.Error(w,"invalid Server status",http.StatusBadGateway);return
  }
  _=json.NewEncoder(w).Encode(result)
 }
}

func main(){
 origin,err:=validatedAPIOrigin(os.Getenv("IFACEMATCH_EXAMPLE_API_ORIGIN"))
 if err!=nil{log.Fatal(err)}
 key,err:=privateAPIKey(os.Getenv("IFACEMATCH_EXAMPLE_API_KEY_FILE"))
 if err!=nil{log.Fatal(err)}
 mux:=http.NewServeMux()
 mux.HandleFunc("/start",startHandler(origin,key))
 mux.HandleFunc("/status",statusHandler(origin,key,issuerClient()))
 mux.HandleFunc("/example.js",func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodGet {http.Error(w,"GET required",http.StatusMethodNotAllowed);return}
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("X-Content-Type-Options","nosniff")
  http.ServeFile(w,r,"example.js")
 })
 mux.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/"||r.Method!=http.MethodGet {http.NotFound(w,r);return}
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("Content-Security-Policy","default-src 'none'; style-src 'unsafe-inline'; script-src 'self'; connect-src 'self'; form-action 'self'")
  http.ServeFile(w,r,"example.html")
 })
 server:=&http.Server{Addr:"127.0.0.1:8099",Handler:mux,
  ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,
  WriteTimeout:20*time.Second,IdleTimeout:30*time.Second,
  MaxHeaderBytes:8192}
 fmt.Fprintln(os.Stdout,"Exemplo iFaceMatch disponível SOMENTE em http://127.0.0.1:8099/")
 log.Fatal(server.ListenAndServe())
}
