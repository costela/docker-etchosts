package main

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_writeEntryWithBanner(t *testing.T) {
	type args struct {
		ip    string
		names []string
	}
	tests := []struct {
		name    string
		args    args
		wantTmp string
		wantErr bool
	}{
		{"do not write empty ip", args{"", []string{"somename", "someothername"}}, "", false},
		{"do not write empty names", args{"1.2.3.4", []string{}}, "", false},
		{"complete entry", args{"1.2.3.4", []string{"somename", "someothername"}}, fmt.Sprintf("%s\n1.2.3.4\tsomename someothername\n", banner), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := &bytes.Buffer{}
			if err := writeEntryWithBanner(tmp, tt.args.ip, tt.args.names); (err != nil) != tt.wantErr {
				t.Errorf("writeEntryWithBanner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTmp := tmp.String(); gotTmp != tt.wantTmp {
				t.Errorf("writeEntryWithBanner() got:\n%#v, want\n%#v", gotTmp, tt.wantTmp)
			}
		})
	}
}
