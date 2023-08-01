// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"time"
	"strconv"
	"net/http"

	log "github.com/sirupsen/logrus"
	"go.amzn.com/lambda/interop"
	"go.amzn.com/lambda/rapidcore"
)

func startHTTPServer(ipport string, sandbox *rapidcore.SandboxBuilder, bs interop.Bootstrap) {
	srv := &http.Server{
		Addr: ipport,
	}

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
		done := make(chan struct{}) // Channel to signal when the response has terminated

		InvokeHandler(w, r, sandbox.LambdaInvokeAPI(), bs, done)

		<-done // Wait until the response has terminated
		// Shutdown the server if we've reached the maximum number of invocations
		maxInvocations--
		if maxInvocations == 0 {
			close(shutdownChan)
		}
	})


	// go routine to handle server shutdown (main thread waits)
	go func() {
		<-shutdownChan
		log.Printf("Maximum invocations (%s) reached. Shutting down the server.", maxInvocationsStr)
		time.Sleep(60 * time.Second) // Grace period for client to receive the response
		if err := srv.Shutdown(nil); err != nil {
			log.Panic(err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Panic(err)
	}

	log.Warnf("Listening on %s", ipport)
}
