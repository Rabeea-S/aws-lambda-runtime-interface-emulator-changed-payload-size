// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"sync"
	"bytes"
	"strconv"
	"net/http"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"go.amzn.com/lambda/interop"
	"go.amzn.com/lambda/rapidcore"
)

func startHTTPServer(ipport string, sandbox *rapidcore.SandboxBuilder, bs interop.Bootstrap) {
	srv := &http.Server{
		Addr: ipport,
	}

	log.Warnf("Listening on %s", ipport)

	maxInvocations := -1 // -1 means unlimited invocations
	// Get max invocations from environment variable
	maxInvocationsStr := os.Getenv("AWS_LAMBDA_SERVER_MAX_INVOCATIONS")
	if maxInvocationsStr != "" {
		if maxInvocationsInt, err := strconv.Atoi(maxInvocationsStr); err == nil {
			maxInvocations = maxInvocationsInt
		} else {
			log.Panicf("Invalid value for AWS_LAMBDA_SERVER_MAX_INVOCATIONS: %s", maxInvocationsStr)
		}
	}

	// Channel to signal server shutdown
	shutdownChan := make(chan struct{})

	// Pass a channel
	http.HandleFunc("/2015-03-31/functions/function/invocations", func(w http.ResponseWriter, r *http.Request) {
		InvokeHandler(w, r, sandbox.LambdaInvokeAPI(), bs, func(invokeResp *ResponseWriterProxy){

			// Forward response if "forward-response" header exists
			if forwardURL := r.Header.Get("forward-response"); forwardURL != "" {
				// Create a wait group to wait for the API request to finish
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					// Marshal the payload to JSON (you can use any other serialization format if needed)
					apiPayloadJSON, err := json.Marshal(string(invokeResp.Body))
					if err != nil {
						log.Errorf("Failed to json marshal API payload: %s", err)
						return
					}

					// Create an API request to the URL in the "forward-response" header
					client := &http.Client{}
					req, err := http.NewRequest("POST", forwardURL, bytes.NewReader(apiPayloadJSON))
					// Add request headers
					req.Header.Add("Authorization", "Token "+os.Getenv("API_ACCESS_KEY"))
					req.Header.Add("Content-Type", "application/json")
					// Send the request
					resp, err := client.Do(req)
					if err != nil {
						log.Errorf("Failed to forward response: %s", err)
						return
					}
					defer resp.Body.Close()
					
					if resp.StatusCode == 200 {
						log.Printf("Forwarded response was successful")
					} else {
						log.Errorf("Forwarding response failed with status code: %d", resp.StatusCode)
					}

					defer wg.Done() // Defer the Done() call to mark the API request as completed
				}()
				
				wg.Wait() // Wait for the API request to finish before proceeding
			}

			// Shutdown the server if the maximum number of invocations is reached
			maxInvocations--
			if maxInvocations == 0 {
				close(shutdownChan)
			}
		})
	})


	// go routine to handle server shutdown (main thread waits)
	go func() {
		<-shutdownChan
		log.Printf("Maximum invocations (%s) reached. Shutting down the server", maxInvocationsStr)
		if err := srv.Shutdown(nil); err != nil {
			log.Panic(err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Panic(err)
	}

}
