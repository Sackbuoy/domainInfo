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

const PORT = ":8000"

type Response struct {
  Status           []string `json:"status"`
  CreatedDate      string   `json:"created`
  ExpirationDate  string   `json:"expiry"`
  Registrar        string   `json:"registrar"`
  Registrant      string   `json:"registrant"`
  ContactEmail    string   `json:"contactEmail"`
}

type ErrorResponse struct {
  Kind    string `json:"kind"`
  Code    int    `json:"code"`
  Message string `json:"message"`
}

const getDomainInfoPath = "/domaininfo/{domain}"
func getDomainInfo(writer http.ResponseWriter, req *http.Request) {
  vars := mux.Vars(req)
  domain := string(vars["domain"])

  writer.Header().Set("Content-Type", "application/json")

  whoisResult, whoisErr := whois.Whois(domain)
  // the whoisparser only supports whois responses for domain names, not IP addresses.
  // If i have time I will implement a second response type for IP addresses as a workaround
  parsedResult, parseErr := whoisparser.Parse(whoisResult) 

  if whoisErr != nil {
    errorMessage := ""
    errorCode := 0
    errorType := ""
    if whoisErr == whois.ErrDomainEmpty {
      writer.WriteHeader(http.StatusBadRequest) 
      errorMessage = "Domain cannot be empty"
      errorCode = 400
      errorType = "bad_request"
    } else {
      writer.WriteHeader(http.StatusInternalServerError) 
      errorMessage = fmt.Sprintf("Failed to fetch WhoisData for %s", domain)
      errorCode = 500
      errorType = "internal_server_error"
    }

    log.Println(errorMessage)

    errResponse := ErrorResponse {
      Kind:      errorType,
      Code:      errorCode,
      Message:   whoisErr.Error(),
    }

    jsonResponse, err := json.Marshal(errResponse)
    if err != nil {
      log.Println("Failed to format WHOIS fetch error as JSON")
      writer.Write([]byte("500 - Internal Server Error"))
    }

    writer.Write(jsonResponse)
  } else if parseErr != nil {
      writer.WriteHeader(http.StatusBadRequest) // 400 Bad request
      errorMessage := fmt.Sprintf("Failed to parse WHOIS response for domain %s", domain)

      log.Println(errorMessage)

      errResponse := ErrorResponse {
        Kind:      "bad_request",
        Code:      400,
        Message:  parseErr.Error(),
      }

      jsonResponse, err := json.Marshal(errResponse)
      if err != nil {
        writer.WriteHeader(http.StatusInternalServerError)
        log.Println("Failed to format parse error as JSON")
        writer.Write([]byte("500 - Internal Server Error"))
      }

      writer.Write(jsonResponse)
  } else {

      response := Response {
        Status:         parsedResult.Domain.Status,
        CreatedDate:     parsedResult.Domain.CreatedDate,
        ExpirationDate: parsedResult.Domain.ExpirationDate,
        Registrar:       parsedResult.Registrar.Name,
        Registrant:      parsedResult.Registrant.Name,
        ContactEmail:   parsedResult.Registrant.Email,
      }
      
      writer.WriteHeader(http.StatusOK)
      jsonResponse, err := json.Marshal(response)
      if err != nil {
        writer.WriteHeader(http.StatusInternalServerError)
        log.Println("Failed to WHOIS response object as JSON")
        writer.Write([]byte("500 - Internal Server Error"))
      }

      writer.Write(jsonResponse)
    }
}

func main() {
  log.Println("Server Started")
  router := mux.NewRouter()
  router.HandleFunc(getDomainInfoPath, getDomainInfo).Methods("GET")

  log.Fatal(http.ListenAndServe(PORT, router))
}