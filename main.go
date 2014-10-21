package main

import (
  "fmt"
  "log"
  "time"
  "errors"
  "net/http"
  "encoding/json"
  "io/ioutil"

  "code.google.com/p/gcfg"
  "github.com/vincent-petithory/dataurl"
  "github.com/goji/httpauth"
  r "github.com/dancannon/gorethink"
)

// Base config fields
type Config struct {
  Port      string
  Root      string
  DbAddress string
  Database  string
  ImageDir  string
  Username  string
  Password  string
}

type ConfigFile struct {
  Server Config
}

// Default config in gcfg format
// Unused parts get assigned 0 according to gcfg
const defaultConfig = `
  [server]
  port = "8080"
  imagedir = "images"
  username = "images"
  password = "dataurltopng"
`

var globalConfig ConfigFile

// Given a filepath, attempt to parse config from file
// If file doesn't exist or parse fails, use default
// Returns any error encountered
func LoadConfiguration(cfgFilepath string) (err error){

  if cfgFilepath != "" {
    err = gcfg.ReadFileInto(&globalConfig, cfgFilepath)
  }

  if err != nil {
    err = gcfg.ReadStringInto(&globalConfig, defaultConfig)
  }

  if err != nil {
    log.Println(err)
  }

  return
}

// Set the route for handling static serving of generated images
// Set the route for POSTing dataurls
// Start the server
func RegisterHandlers(){
  http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(globalConfig.Server.ImageDir))))
  http.Handle("/stringtopng/", httpauth.SimpleBasicAuth(globalConfig.Server.Username, globalConfig.Server.Password)(http.HandlerFunc(stringToPngHandler)))
  log.Fatal(http.ListenAndServe(":"+globalConfig.Server.Port, nil))
}

func main(){
  err := LoadConfiguration("Config.gcfg")

  if err != nil {
    log.Println(err.Error())
  }

  fmt.Printf("Doc Root: %s\nListen On: :%s\n", globalConfig.Server.Root, globalConfig.Server.Port)

  RegisterHandlers()
}

type MakeImageRequest struct{
  Sitename    string
  Dataurl     string
}

type MakeImageResponse struct{
  Success   bool
  Filepath  string
}

// Handles "/stingtopng/" POST requests
// Attempt to save png and return a json object with the static url
func stringToPngHandler(w http.ResponseWriter, r *http.Request){
  if r.Method != "POST" {
    http.Error(w, r.Method+" not allowed", http.StatusMethodNotAllowed)
    return
  }

  //_ := InitDB()

  req := MakeImageRequest{}
  dec := json.NewDecoder(r.Body)
  err := dec.Decode(&req)

  filepath, err := stringToPngFile(req)
  defer r.Body.Close()

  if err != nil {
    http.Error(w, "Bad request", http.StatusBadRequest)
    return
  }

  response := MakeImageResponse{true, filepath}
  json, err := json.Marshal(response)

  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  w.Header().Set("Access-Control-Allow-Origin", "*")
  w.Header().Set("Content-type", "application/json")
  w.Write(json)
}

// Uses dataurl to convert the dataurl to a file
// Filename is [Sitename]-[current timestamp].png
// Returns the full filepath and any encountered errors
func stringToPngFile(m MakeImageRequest) (filepath string, err error){
  dataURL, err := dataurl.DecodeString(m.Dataurl)

  if err != nil {
    log.Println(err)
    return
  }

  t := time.Now()
  filename := m.Sitename+"-"+t.Format("20060102150405")+".png"

  if dataURL.ContentType() == "image/png" {
    filepath = globalConfig.Server.ImageDir+"/"+filename
    err = ioutil.WriteFile(filepath, dataURL.Data, 0644)
    if err != nil {
      log.Println(err)
    }
  } else {
    err = errors.New("Invalid filetype")
  }

  return
}

// Initialize the database session so we can index generated files
func InitDB() *r.Session {

  session, err := r.Connect(r.ConnectOpts{
    Address:  globalConfig.Server.DbAddress, // "localhost:28015"
    Database: globalConfig.Server.Database,
  })
  if err != nil {
    fmt.Printf("No database connection. Conversions will not be logged.")
  }

  err = r.DbCreate(globalConfig.Server.Database).Exec(session)
  if err != nil {
    fmt.Println(err)
  }

  _, err = r.Db(globalConfig.Server.Database).TableCreate("images").RunWrite(session)
  if err != nil {
    fmt.Println(err)
  }

  return session
}
