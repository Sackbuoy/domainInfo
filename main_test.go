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

  // store valid function calls so i can reset mocks later
  validWhoisErrToJson = whoisErrToJson
  validResponseToJson = responseToJson
  validFetchWhoIs = fetchWhoIs
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
  domain := "goopy.us" // I own this domain, so I know it is valid

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


// test for when formatting an erroneous whois response as JSON failed(requires mocking json.marshal)
func TestFailedToFormatErrAsJson(t *testing.T) {
  domain := ""

  // this is for suppressing log output and saving it to check later
  var logBuf bytes.Buffer
  log.SetOutput(&logBuf)
  defer func() {
      log.SetOutput(os.Stderr)
  }()

  // mocks the toJson variable, forcing it to return an error
  whoisErrToJson = func(err WhoIsError) ([]byte, error) {
    return nil, fmt.Errorf("marshal err")
  }

  response := sendDomainInfoRequest(domain, t)
  logOutput := logBuf.String()

  expected := 500
  checkStatusCode(response.Code, expected, t)

  expectedMessage := "Failed to parse WHOIS response object as JSON"
  checkLogOutput(expectedMessage, logOutput, t)


  expectedResponse := "500 - Internal Server Error"
  actualResponse := response.Body.String()
  if actualResponse != expectedResponse {
    t.Fatal(formatExpectedVsActual(expectedResponse, actualResponse))
  }

  // reset mocks when im done
  whoisErrToJson = validWhoisErrToJson
}


// test for when formatting a valid whois response as JSON failed(requires mocking json.marshal)
func TestFailedToFormatResponseAsJson(t *testing.T) {
  domain := "goopy.us"

  // this is for suppressing log output and saving it to check later
  var logBuf bytes.Buffer
  log.SetOutput(&logBuf)
  defer func() {
      log.SetOutput(os.Stderr)
  }()

  responseToJson = func(resp Response) ([]byte, error) {
    return nil, fmt.Errorf("marshal err")
  }

  response := sendDomainInfoRequest(domain, t)
  logOutput := logBuf.String()

  expected := 500
  checkStatusCode(response.Code, expected, t)

  expectedMessage := "Failed to parse WHOIS response object as JSON"
  checkLogOutput(expectedMessage, logOutput, t)


  expectedResponse := "500 - Internal Server Error"
  actualResponse := response.Body.String()
  if actualResponse != expectedResponse {
    t.Fatal(formatExpectedVsActual(expectedResponse, actualResponse))
  }

  // reset mocks when im done
  responseToJson = validResponseToJson
}


// test for when whois call failed for a non-user related reason
func TestWhoisCallFailure(t *testing.T) {
  domain := "goopy.us"

  // this is for suppressing log output and saving it to check later
  var logBuf bytes.Buffer
  log.SetOutput(&logBuf)
  defer func() {
      log.SetOutput(os.Stderr)
  }()

  whoIsErrorMsg := "whois: no whois server found for domain" // this is the only other error it can throw

  fetchWhoIs = func(domain string) (string, error) {
    return "", errors.New("whois: no whois server found for domain")
  }

  response := sendDomainInfoRequest(domain, t)
  logOutput := logBuf.String()

  expected := 500
  checkStatusCode(response.Code, expected, t)

  checkLogOutput(whoIsErrorMsg, logOutput, t)

  if response.Body == nil {
    t.Fatal("Got no response body from WHOIS call")
  }

  // reset mocks when im done
  fetchWhoIs = validFetchWhoIs
}


/* HELPER METHODS */

func formatExpectedVsActual(expected string, actual string) (string) {
  return fmt.Sprintf("\nExpected:\n  %s\nto contain:\n  %s\n", actual, expected)
}

func findExpectedMessage(domain string) (message string) {
  if strings.TrimSpace(domain) == "" {
    return "whois: domain is empty" // this is copied directly from the whois output
  } else {
    return "whoisparser: domain whois data is invalid" // this is copied directly from the whois-parser output
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
  handler := http.HandlerFunc(getDomainInfoHandler)
  handler.ServeHTTP(responseRecorder, req)
  return responseRecorder
}

func checkStatusCode(actual int, expected int, t *testing.T) {
  if actual != expected {
    errorMsg := formatExpectedVsActual(strconv.Itoa(expected), strconv.Itoa(actual))
    t.Fatal(errorMsg)
  }
}