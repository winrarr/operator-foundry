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

package controller

import (
	"reflect"
	"testing"
)

func TestParseWatchNamespaces(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "empty means all namespaces", raw: "", want: nil},
		{name: "whitespace means all namespaces", raw: "  ", want: nil},
		{name: "trims and sorts", raw: "team-b, team-a", want: []string{"team-a", "team-b"}},
		{name: "rejects empty entry", raw: "team-a,,team-b", wantErr: true},
		{name: "rejects duplicate", raw: "team-a,team-a", wantErr: true},
		{name: "rejects invalid name", raw: "Team-A", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseWatchNamespaces(test.raw)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseWatchNamespaces(%q) error = %v, wantErr %t", test.raw, err, test.wantErr)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ParseWatchNamespaces(%q) = %#v, want %#v", test.raw, got, test.want)
			}
		})
	}
}

func TestCacheOptionsForWatchNamespaces(t *testing.T) {
	options, err := CacheOptionsForWatchNamespaces("team-a,team-b")
	if err != nil {
		t.Fatalf("CacheOptionsForWatchNamespaces() error = %v", err)
	}
	if len(options.DefaultNamespaces) != 2 {
		t.Fatalf("DefaultNamespaces length = %d, want 2", len(options.DefaultNamespaces))
	}
	for _, namespace := range []string{"team-a", "team-b"} {
		if _, ok := options.DefaultNamespaces[namespace]; !ok {
			t.Errorf("DefaultNamespaces is missing %q", namespace)
		}
	}

	unscoped, err := CacheOptionsForWatchNamespaces("")
	if err != nil {
		t.Fatalf("CacheOptionsForWatchNamespaces(\"\") error = %v", err)
	}
	if unscoped.DefaultNamespaces != nil {
		t.Fatalf("empty scope DefaultNamespaces = %#v, want nil", unscoped.DefaultNamespaces)
	}
}
