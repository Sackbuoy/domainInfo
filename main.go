package main

import (
  "fmt"
  "log"
  "net/http"
  "github.com/gorilla/mux"
  "github.com/likexian/whois"
  "github.com/likexian/whois-parser"
  "encoding/json"
)

const PORT = ":8080"

type Response struct {
  Whois      WhoIsInfo `json:"whoisinfo"`
}

type WhoIsInfo struct {
  Status          []string `json:"status"`
  CreatedDate     string   `json:"created`
  ExpirationDate  string   `json:"expiry"`
  Registrar       string   `json:"registrar"`
  Registrant      string   `json:"registrant"`
  ContactEmail    string   `json:"contactEmail"`
}

type WhoIsError struct {
  Kind    string `json:"kind"`
  Code    int    `json:"code"`
  Message string `json:"message"`
}

// Handler for the "domaininfo/{domain}" path
// This handler will call all information retrievers(e.g Whois)
const getDomainInfoPath = "/domaininfo/{domain}"
func getDomainInfoHandler(writer http.ResponseWriter, req *http.Request) {
  req.Header.Add("Accept-Charset","utf-8")
  writer.Header().Set("Content-Type", "application/json")
  vars := mux.Vars(req)
  domain := string(vars["domain"])

  whoisInfo, whoisErr := getWhoIsInfo(domain)
  if whoisErr != nil { // check for errors from all retrievers here
    log.Println(whoisErr.Message)

    jsonResponse, parseErr := whoisErrToJson(*whoisErr)
    if parseErr != nil {
      log.Println("Failed to parse WHOIS response object as JSON")
      internalServerError(writer, parseErr)
    } else {
      switch whoisErr.Code {
      case 400:
        writer.WriteHeader(http.StatusBadRequest) 
        writer.Write(jsonResponse)
      default:
        writer.WriteHeader(http.StatusInternalServerError)
        writer.Write(jsonResponse)
      }
    }

  } else {
    response := Response{Whois: *whoisInfo}

    jsonResponse, parseErr := responseToJson(response)
    if parseErr != nil {
      log.Println("Failed to parse WHOIS response object as JSON")
      internalServerError(writer, parseErr)
    } else {
      writer.WriteHeader(http.StatusOK)
      writer.Write(jsonResponse)
    }
  }
}


// Retriever for whois data
func getWhoIsInfo(domain string) (*WhoIsInfo, *WhoIsError) {
  whoisResult, whoisErr := fetchWhoIs(domain)
  parsedResult, parseErr := whoisparser.Parse(whoisResult) 

  // check for error from whois
  if whoisErr != nil { 
    errorKind := "bad_request"
    errorCode := 400
    errorMessage := fmt.Sprintf("WHOIS call with domain '%s' failed with error: %s", domain, whoisErr.Error())

    if whoisErr != whois.ErrDomainEmpty {
      log.Printf("Failed to fetch WhoisData for %s", domain)
      log.Println(whoisErr)
      errorKind = "internal_server_error"
      errorCode = 500
    }

    errResponse := WhoIsError {
      Kind:      errorKind,
      Code:      errorCode,
      Message:   errorMessage,
    }

    return nil, &errResponse
  } 

  // check for error from whois-parser
  if parseErr != nil {
    errorMessage := fmt.Sprintf("WHOIS parser with domain '%s' failed with error: %s", domain, parseErr.Error())
    errResponse := WhoIsError {
      Kind:      "bad_request",
      Code:      400,
      Message:  errorMessage,
    }

    return nil, &errResponse
  } 

  // no errors found,
  response := WhoIsInfo {
    Status:         parsedResult.Domain.Status,
    CreatedDate:     parsedResult.Domain.CreatedDate,
    ExpirationDate: parsedResult.Domain.ExpirationDate,
    Registrar:       parsedResult.Registrar.Name,
    Registrant:      parsedResult.Registrant.Name,
    ContactEmail:   parsedResult.Registrant.Email,
  }

  return &response, nil 
}

// putting these functions in vars like this makes tesing a lot easier
var fetchWhoIs = func(domain string) (string, error) {
  return whois.Whois(domain)
}

var whoisErrToJson = func(err WhoIsError) ([]byte, error) {
  return json.Marshal(err)
}

var responseToJson = func(resp Response) ([]byte, error) {
  return json.Marshal(resp)
}

func internalServerError(writer http.ResponseWriter, err error) {
  log.Println(err)
  writer.WriteHeader(http.StatusInternalServerError)
  writer.Header().Set("Content-Type", "text/plain")
  writer.Write([]byte("500 - Internal Server Error"))
}

const rootPath = "/"
func rootHandler(writer http.ResponseWriter, req *http.Request) {
  req.Header.Add("Accept-Charset","utf-8")
  writer.Header().Set("Content-Type", "text/plain")
  writer.Write([]byte("DomainInfo API application"))
}

func main() {
  log.Println("Server Started")
  router := mux.NewRouter()

  // Handlers
  router.HandleFunc(getDomainInfoPath, getDomainInfoHandler).Methods("GET")
  router.HandleFunc(rootPath, rootHandler).Methods("GET")

  log.Fatal(http.ListenAndServe(PORT, router))
}