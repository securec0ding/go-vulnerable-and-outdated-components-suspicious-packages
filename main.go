package main

import (
  "fmt"
  "html/template"
  "io"
  "log"
  "net/http"
  "net/http/httptest"
  "net/http/httputil"
  "os"
  "path/filepath"
  "strings"
  "github.com/mholt/archiver"
)

const UPLOADS = "uploads/"

type Response struct {
  Files []string
}

func main() {
  openLogFile("/home/whitehat/access.log")

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    response := Response{}

    // Walk existing uploads
    filepath.Walk(UPLOADS, func(path string, info os.FileInfo, err error) error {
      if err == nil && !info.IsDir() {
        fileName := strings.TrimPrefix(path, UPLOADS)
        if string(fileName[0]) != "." {
          // Add non-hidden files to list of existing uploads
          response.Files = append(response.Files, fileName)
        }
      }
      return nil
    })

    if r.Method == "POST" {
      // Read archive from uploaded file
      fileReader, handler, err := r.FormFile("zip")
      if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }
      defer fileReader.Close()

      zipFilePath := UPLOADS + handler.Filename

      // Copy archive file contents into new file
      file, _ := os.OpenFile(zipFilePath, os.O_WRONLY|os.O_CREATE, 0666)
      defer file.Close()
      io.Copy(file, fileReader)

      // Unzip archive contents into UPLOADS directory
      err = archiver.Zip.Open(zipFilePath, UPLOADS)
      os.Remove(zipFilePath)

      if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }
    }

    tmpl := template.Must(template.ParseFiles("index.html"))
    tmpl.Execute(w, response)
  })

  http.ListenAndServe(":"+os.Getenv("VIRTUAL_PORT"), logHandler(http.DefaultServeMux))
}

func logHandler(handler http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    x, err := httputil.DumpRequest(r, true)
    if err != nil {
      http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
      return
    }
    log.Println(fmt.Sprintf("Q %q", x))
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, r)
    log.Println(fmt.Sprintf("A %d", rec.Code))

    handler.ServeHTTP(w, r)
  })
}

func openLogFile(logfile string) {
  lf, _ := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
  log.SetOutput(lf)
  log.SetFlags(log.Ltime)
}