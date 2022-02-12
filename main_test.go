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
)

const baseURL = "http://0.0.0.0/domaininfo"


// Happy path with a valid domain, and no error logs
func TestGetDomainInfo(t *testing.T) {
  domain := "whoiswrapper.com"

  // this is for suppressing log output and saving it to check later
  var logBuf bytes.Buffer
  log.SetOutput(&logBuf)
  defer func() {
      log.SetOutput(os.Stderr)
  }()

  response := sendRequest(domain, t)
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
      
      response := sendRequest(testTable.domain, t)
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

func findExpectedMessage(domain string) (message string) {
  if strings.TrimSpace(domain) == "" {
    return "Domain cannot be empty"
  } else {
    return fmt.Sprintf("Failed to parse WHOIS response for domain %s", domain)
  }
}

func checkLogOutput(expected string, actual string, t *testing.T) {
  if !strings.Contains(actual, expected) {
    t.Fatal(fmt.Sprintf("Expected: \n %s to contain: \n %s", actual, expected))
  }
}

func sendRequest(domain string, t *testing.T) *httptest.ResponseRecorder {
  req, err := http.NewRequest("GET", baseURL, nil)
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

func checkStatusCode(code int, expected int, t *testing.T) {
  if code != expected {
    errorMsg := fmt.Sprintf("Expected status code %d but got %d", expected, code)
    t.Fatal(errorMsg)
  }
}