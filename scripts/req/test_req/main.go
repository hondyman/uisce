//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/api/catalog/nodes?tenant_id=99e99e99-99e9-49e9-89e9-99e99e99e999&tenant_instance_id=25b5dce3-27d9-4773-933e-6ee29a42871f&type=business_term&limit=1", nil)
	// bypass auth by not sending token, wait, it might 401.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nBody: %s\n", resp.StatusCode, string(body))
}
