package xraycfg

import (
	"encoding/json"
	"testing"
)

func TestWithEmailsKeepsExistingAndFillsID(t *testing.T) {
	in := `{"clients":[{"id":"abcdef1234567890","flow":""},{"id":"bbbbbbbb","email":"kept"}]}`
	out := WithEmails(in, 7)
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	clients := doc["clients"].([]interface{})
	first := clients[0].(map[string]interface{})
	second := clients[1].(map[string]interface{})
	if first["email"] != "abcdef1234567890" {
		t.Fatalf("first email %v", first["email"])
	}
	if second["email"] != "kept" {
		t.Fatalf("second email %v", second["email"])
	}
	if UserLabel(7, in, "abcdef1234567890") != "abcdef12" {
		t.Fatalf("label %s", UserLabel(7, in, "abcdef1234567890"))
	}
	if UserLabel(7, in, "kept") != "kept" {
		t.Fatal(UserLabel(7, in, "kept"))
	}
}

func TestWithEmailsPasswordIsNotThePassword(t *testing.T) {
	in := `{"clients":[{"password":"secret-pass"}]}`
	out := WithEmails(in, 3)
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	email := doc["clients"].([]interface{})[0].(map[string]interface{})["email"].(string)
	if email == "secret-pass" || email == "" {
		t.Fatal(email)
	}
	if UserLabel(3, in, email) != "用户1" {
		t.Fatal(UserLabel(3, in, email))
	}
}

func TestWithSniffingRespectsDisabled(t *testing.T) {
	in := `{"enabled":false}`
	if WithSniffing(in) != in {
		t.Fatal(WithSniffing(in))
	}
	out := WithSniffing("")
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["enabled"] != true {
		t.Fatal(doc)
	}
}
