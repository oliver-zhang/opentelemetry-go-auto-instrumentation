// Copyright (c) 2024 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"os"
	"strconv"
)

var logger = zerolog.New(os.Stdout).With().
	Timestamp().
	Str("role", "my-service").
	Str("host", "127.0.0.1").
	Logger()

var port int

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/b")
	if err != nil {
		logger.Printf("request provider error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger.Print("abcde")
	_, _ = w.Write([]byte("success"))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}
