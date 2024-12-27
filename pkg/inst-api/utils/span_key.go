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

package utils

import "go.opentelemetry.io/otel/attribute"

const DB_CLIENT_KEY = attribute.Key("opentelemetry-traces-span-key-db-client")

const RPC_SERVER_KEY = attribute.Key("opentelemetry-traces-span-key-rpc-server")
const RPC_CLIENT_KEY = attribute.Key("opentelemetry-traces-span-key-rpc-client")
const HTTP_CLIENT_KEY = attribute.Key("opentelemetry-traces-span-key-http-client")
const HTTP_SERVER_KEY = attribute.Key("opentelemetry-traces-span-key-http-server")

const PRODUCER_KEY = attribute.Key("opentelemetry-traces-span-key-producer")
const CONSUMER_RECEIVE_KEY = attribute.Key("opentelemetry-traces-span-key-consumer-receive")
const CONSUMER_PROCESS_KEY = attribute.Key("opentelemetry-traces-span-key-consumer-process")

const KIND_SERVER = attribute.Key("opentelemetry-traces-span-key-kind-server")
const KIND_CLIENT = attribute.Key("opentelemetry-traces-span-key-kind-client")
const KIND_CONSUMER = attribute.Key("opentelemetry-traces-span-key-kind-consumer")
const KIND_PRODUCER = attribute.Key("opentelemetry-traces-span-key-kind-producer")

const OTEL_CONTEXT_KEY = attribute.Key("opentelemetry-http-server-route-key")

