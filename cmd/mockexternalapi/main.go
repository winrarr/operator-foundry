/*
Copyright 2026 winrarr.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/winrarr/operator-foundry/internal/mockexternalapi"
)

func main() {
	port := flag.Int("port", 8080, "HTTP port")
	flag.Parse()

	address := fmt.Sprintf(":%d", *port)
	log.Printf("starting disposable mock external API on %s", address)
	if err := http.ListenAndServe(address, mockexternalapi.New().Handler()); err != nil {
		log.Fatal(err)
	}
}
