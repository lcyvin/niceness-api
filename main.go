package main

import (
	"fmt"
	"net/http"
	"os"
  "time"
	"strconv"
	"strings"
  "encoding/json"

	"github.com/shirou/gopsutil/v3/process"
)

const (
  DEFAULT_PORT string = "3030"
  NICE_ROUTE_SLUG_PATH string = "/nice/"
)

type HealthCheck struct {
  Status string `json:"status"`
  Time int64 `json:"timestamp"`
}

func NewHealthCheck() *HealthCheck {
  hc := &HealthCheck{
    Status: "OK",
    Time: time.Now().Unix(),
  }

  return hc
}

// just a little DRY
func ServerError(e error, w http.ResponseWriter) {
  http.Error(w, fmt.Sprint(e), 502)
}

// ProcStat contains a PID and its assosciated Nice value
type ProcStat struct {
  Pid int32 `json:"PID"`
  Niceness int32 `json:"Nice"`
}

// GetNiceness will return a Nice value for a given valid PID, assuming said PID exists
func GetNiceness(pid int32) (*ProcStat, error) {
  ps := &ProcStat{Pid: pid}

  proc, err := process.NewProcess(ps.Pid)
  if err != nil {
    return nil, err
  }

  niceness, err := proc.Nice()
  if err != nil {
    return nil, err
  }

  ps.Niceness = niceness

  return ps, nil
}

// NicenessHandler either responds with the server itself's PID and Nice 
// or takes an optional slug which returns the information on that particular
// process, if it exists
func NicenessHandler(w http.ResponseWriter, r *http.Request) {
  var slug string
  var pid int32
  var ps *ProcStat

  if strings.HasPrefix(r.URL.Path, NICE_ROUTE_SLUG_PATH) {
    slug = r.URL.Path[len(NICE_ROUTE_SLUG_PATH):]
  }

  if slug != "" {
    slugInt, err := strconv.Atoi(slug)
    if err != nil {
      ServerError(err, w)
      return
    }

    pid = int32(slugInt)
  } else {
    pid = int32(os.Getpid())
  }

  ps, err := GetNiceness(pid)
  if err != nil {
    ServerError(err, w)
    return
  }

  out, err := json.Marshal(ps)
  if err != nil {
    ServerError(err, w)
    return
  }

  w.Write(out)
}

// RootHandler basically exists as a liveness probe separate from 
// the readiness probe test against /nice
func RootHandler(w http.ResponseWriter, r *http.Request) {
  hc := NewHealthCheck()
  j, err := json.Marshal(hc)
  if err != nil {
    ServerError(err, w)
  }
  
  w.Write(j)
}

func main() {
  port := os.Getenv("API_PORT")
  if port == "" {
    port = DEFAULT_PORT
  }

  fmt.Printf("Starting server on port %s\n", port)

  // using something like the mux package would make this nicer,
  // but I want to avoid adding opinionated dependencies for this
  // particular exercise
  http.HandleFunc("/nice", NicenessHandler)
  http.HandleFunc(NICE_ROUTE_SLUG_PATH, NicenessHandler)
  http.HandleFunc("/", RootHandler)
  http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
