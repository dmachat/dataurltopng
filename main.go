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
  r "github.com/dancannon/gorethink"
)

type Config struct {
  Port int
  Root string
  ImageDir string
  Username string
  Password string
}

type ConfigFile struct {
  Server Config
}

const defaultConfig = `
  [server]
  port = 8080
  imagedir = "images"
  username = "chartbuilder"
  password = "chartbuilder"
`

var globalConfig *ConfigFile

func LoadConfiguration(cfgFile string) (cfg ConfigFile, err error){

  if cfgFile != "" {
    err = gcfg.ReadFileInto(&cfg, cfgFile)
  }

  if err != nil {
    err = gcfg.ReadStringInto(&cfg, defaultConfig)
  }

  if err != nil {
    log.Fatal(err)
  }

  return
}

func RegisterHandlers(cfg ConfigFile){
  http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(cfg.Server.ImageDir))))
  http.HandleFunc("/stringtopng/", stringToPngHandler)
  log.Fatal(http.ListenAndServe(":8080", nil))
}

func main(){
  config, err := LoadConfiguration("Config.gcfg")

  if err != nil {
    log.Fatalln(err.Error())
  }

  fmt.Printf("Doc Root: %s\nListen On: :%d\n", config.Server.Root, config.Server.Port)

  RegisterHandlers(config)
}

type MakeImageRequest struct{
  Sitename    string
  Dataurl     string
}

type MakeImageResponse struct{
  Success   bool
  Filepath  string
}

func stringToPngHandler(w http.ResponseWriter, r *http.Request){
  if r.Method != "POST" {
    http.Error(w, r.Method+" not allowed", http.StatusMethodNotAllowed)
    return
  }

  session := InitDB(*cfg)

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

func stringToPngFile(m MakeImageRequest) (filepath string, err error){
  dataURL, err := dataurl.DecodeString(m.Dataurl)

  if err != nil {
    log.Fatal(err)
    return
  }

  t := time.Now()
  filename := m.Sitename+"-"+t.Format("20060102150405")+".png"

  if dataURL.ContentType() == "image/png" {
    err = ioutil.WriteFile(filename, dataURL.Data, 0644)
    if err != nil {
      log.Fatal(err)
    }
    filepath = filename
  } else {
    err = errors.New("Invalid filetype")
  }

  return
}

func InitDB(cfg ConfigFile) *r.Session {

  session, err := r.Connect(r.ConnectOpts{
    Address:  cfg.Server.dbAddress, // "localhost:28015"
    Database: cfg.Server.Database,
  })
  if err != nil {
    fmt.Printf("No database connection. Conversions will not be logged.")
  }

  err = r.DbCreate(cfg.Server.Database).Exec(session)
  if err != nil {
    fmt.Println(err)
  }

  _, err = r.Db(cfg.Server.Database).TableCreate("images").RunWrite(session)
  if err != nil {
    fmt.Println(err)
  }

  return session
}
