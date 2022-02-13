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
  Domain      DomainInfo `json:"domaininfo"`
}

type DomainInfo struct {
  Status          []string `json:"status"`
  CreatedDate     string   `json:"created`
  ExpirationDate  string   `json:"expiry"`
  Registrar       string   `json:"registrar"`
  Registrant      string   `json:"registrant"`
  ContactEmail    string   `json:"contactEmail"`
}

type DomainErrorResponse struct {
  Kind    string `json:"kind"`
  Code    int    `json:"code"`
  Message string `json:"message"`
}

const getDomainInfoPath = "/domaininfo/{domain}"
func getDomainInfoHandler(writer http.ResponseWriter, req *http.Request) {
  req.Header.Add("Accept-Charset","utf-8")
  writer.Header().Set("Content-Type", "application/json")
  vars := mux.Vars(req)
  domain := string(vars["domain"])

  domainInfo, domainErr := getDomainInfo(writer, domain)
  if domainErr != nil { // add checks for other domain info calls here
    log.Println(domainErr.Message)


    jsonResponse, parseErr := json.Marshal(domainErr)
    if parseErr != nil {
      log.Fatal("Failed to parse WHOIS response object as JSON")
      internalServerError(writer, parseErr)
    } 

    switch domainErr.Code {
    case 400:
      writer.WriteHeader(http.StatusBadRequest) 
      writer.Write(jsonResponse)
    default:
      writer.WriteHeader(http.StatusInternalServerError)
      writer.Write(jsonResponse)
    }

  } else {
    response := Response{Domain: *domainInfo}

    jsonResponse, parseErr := json.Marshal(response)
    if parseErr != nil {
      log.Fatal("Failed to parse WHOIS response object as JSON")
      internalServerError(writer, parseErr)
    } else {
      writer.WriteHeader(http.StatusOK)
      writer.Write(jsonResponse)
    }
  }
}


func getDomainInfo(writer http.ResponseWriter, domain string) (*DomainInfo, *DomainErrorResponse) {
  whoisResult, whoisErr := whois.Whois(domain)
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

    errResponse := DomainErrorResponse {
      Kind:      errorKind,
      Code:      errorCode,
      Message:   errorMessage,
    }

    return nil, &errResponse
  } 

  // check for error from whois-parser
  if parseErr != nil {
    errorMessage := fmt.Sprintf("WHOIS parser with domain '%s' failed with error: %s", domain, parseErr.Error())
    errResponse := DomainErrorResponse {
      Kind:      "bad_request",
      Code:      400,
      Message:  errorMessage,
    }

    return nil, &errResponse
  } 

  // no errors found,
  response := DomainInfo {
    Status:         parsedResult.Domain.Status,
    CreatedDate:     parsedResult.Domain.CreatedDate,
    ExpirationDate: parsedResult.Domain.ExpirationDate,
    Registrar:       parsedResult.Registrar.Name,
    Registrant:      parsedResult.Registrant.Name,
    ContactEmail:   parsedResult.Registrant.Email,
  }

  return &response, nil
    
}

func internalServerError(writer http.ResponseWriter, err error) {
  log.Fatal(err)
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
  router.HandleFunc(getDomainInfoPath, getDomainInfoHandler).Methods("GET")
  router.HandleFunc(rootPath, rootHandler).Methods("GET")

  log.Fatal(http.ListenAndServe(PORT, router))
}