package main

import (
  "fmt"
  "testing"
  "net/http"
  "net/http/httptest"
  "github.com/gorilla/mux"
  "log"
  "bytes"
  "os"
  "strings"
  "encoding/json"
  "errors"
  "strconv"
)

var (
  root          = fmt.Sprintf("http://0.0.0.0:%s", PORT)
  domainInfoURL = fmt.Sprintf("%s/%s", root, "domaininfo")
  jsonMarshal = json.Marshal
)


func TestRoot(t *testing.T) {
  req, err := http.NewRequest("GET", root, nil)
  if err != nil {
    t.Fatal(err)
  }

  responseRecorder := httptest.NewRecorder()
  handler := http.HandlerFunc(rootHandler)
  handler.ServeHTTP(responseRecorder, req)

  expected := "DomainInfo API application"
  actual := responseRecorder.Body.String()
  if (actual != expected) {
    t.Fatal(formatExpectedVsActual(expected, actual))
  }
}


// Happy path with a valid domain, and no error logs
func TestGetDomainInfo(t *testing.T) {
  domain := "whoiswrapper.com"

  // this is for suppressing log output and saving it to check later
  var logBuf bytes.Buffer
  log.SetOutput(&logBuf)
  defer func() {
      log.SetOutput(os.Stderr)
  }()

  response := sendDomainInfoRequest(domain, t)
  logOutput := logBuf.String()

  expected := 200
  checkStatusCode(response.Code, expected, t)

  if !(len(logOutput) == 0) {
    t.Fatal(fmt.Sprintf("Expected no log output but got %s", logOutput))
  }
  
  if response.Body == nil {
    t.Fatal("Got no response body from WHOIS call")
  }
}


// Templated tests for nonexistent, malformatted, or empty/blank domain names
// with a nonexistent domain name, it fails to parse the Whois response, 
// with a blank/empty domain name, the whois request fails
func TestBadDomain(t *testing.T) {
  var tests = []struct {
    domain string
  }{
    {"bad_domain"},
    {"!@#$"},
    {" "},
    {""},
  }

  for _, testTable := range tests {
    testName := fmt.Sprintf("%s", testTable.domain)
    t.Run(testName, func(t *testing.T) {

      // this is for suppressing log output and saving it to check later
      var logBuf bytes.Buffer
      log.SetOutput(&logBuf)
      defer func() {
          log.SetOutput(os.Stderr)
      }()
      
      response := sendDomainInfoRequest(testTable.domain, t)
      logOutput := logBuf.String()

      expected := 400
      checkStatusCode(response.Code, expected, t)

      expectedMessage := findExpectedMessage(testTable.domain)
      checkLogOutput(expectedMessage, logOutput, t)

      if response.Body == nil {
        t.Fatal("Got no response body from WHOIS call")
      }
    })
  }
}


/* HELPER METHODS */

func formatExpectedVsActual(expected string, actual string) (string) {
  return fmt.Sprintf("Expected:\n  %s\nto equal:\n  %s", actual, expected)
}

func findExpectedMessage(domain string) (message string) {
  if strings.TrimSpace(domain) == "" {
    return "Domain cannot be empty"
  } else {
    return fmt.Sprintf("Failed to parse WHOIS response for domain %s", domain)
  }
}

func checkLogOutput(expected string, actual string, t *testing.T) {
  if !strings.Contains(actual, expected) {
    t.Fatal(formatExpectedVsActual(expected, actual))
  }
}

func sendDomainInfoRequest(domain string, t *testing.T) *httptest.ResponseRecorder {
  req, err := http.NewRequest("GET", domainInfoURL, nil)
  if err != nil {
    t.Fatal(err)
  }

  vars := map[string]string{
    "domain": domain,
  }
  req = mux.SetURLVars(req, vars)

  responseRecorder := httptest.NewRecorder()
  handler := http.HandlerFunc(getDomainInfo)
  handler.ServeHTTP(responseRecorder, req)
  return responseRecorder
}

func checkStatusCode(actual int, expected int, t *testing.T) {
  if actual != expected {
    errorMsg := formatExpectedVsActual(strconv.Itoa(expected), strconv.Itoa(actual))
    t.Fatal(errorMsg)
  }
}

func fakemarshal(v interface{}) ([]byte, error) {
    return []byte{}, errors.New("Marshalling failed")
}

func restoremarshal(replace func(v interface{}) ([]byte, error)) {
    jsonMarshal = replace
}