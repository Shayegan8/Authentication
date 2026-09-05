package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var l = fmt.Println

func main() {

	// /register GET,POST

	respak, _ := http.Get("http://127.0.0.1:1234/token")
	bufjerk, _ := io.ReadAll(respak.Body)
	respak.Body.Close()
	l(string(bufjerk))
	bucket1, _ := http.NewRequest("GET", "http://127.0.0.1:1235/login", nil)
	bucket1.Header.Set("sideline", string(bufjerk))
	client0 := &http.Client{}
	// GET
	resp0, _ := client0.Do(bucket1)
	var buffesag []byte
	resp0.Body.Read(buffesag)
	l(string(buffesag))
	bucket1Cookie := resp0.Cookies()[0]
	vals := strings.Split(bucket1Cookie.Value, ",")
	csrf_token := vals[1]
	l("/login GET status", resp0.Status)

	// POST
	toRegister, _ := http.NewRequest("POST", "http://127.0.0.1:1235/login", nil)
	toRegister.AddCookie(bucket1Cookie)

	toRegister.Header.Set("csrf-token", csrf_token)
	client := &http.Client{}
	toRegisterResponse, _ := client.Do(toRegister)
	l("/login POST status", toRegisterResponse.Status)

	captchaGeneratedCookie := toRegisterResponse.Cookies()[0]
	// /register/validation GET,POST

	respak1, _ := http.Get("http://127.0.0.1:1234/token")
	bufjerk1, _ := io.ReadAll(respak1.Body)
	respak1.Body.Close()

	// GET
	bucket2, _ := http.NewRequest("GET", "http://127.0.0.1:1235/login/validation", nil)
	client00 := &http.Client{}
	bucket2.Header.Set("sideline", string(bufjerk1))
	resp00, _ := client00.Do(bucket2)
	bucket2Cookie := resp00.Cookies()[0]
	vals = strings.Split(bucket2Cookie.Value, ",")
	csrf_token = vals[1]
	l("/login/validation GET status", resp00.Status)

	var x string
	var y string
	fmt.Println("Please enter the result with order of first x and y: ")
	fmt.Scan(&x, &y)

	// POST

	toRegisterValidation, _ := http.NewRequest("POST", "http://127.0.0.1:1235/login/validation", nil)
	println("hereee")
	toRegisterValidation.AddCookie(bucket2Cookie)
	println("hereee2")
	toRegisterValidation.AddCookie(captchaGeneratedCookie)

	toRegisterValidation.Header.Set("content-type", "application/json")

	captchaJSON := fmt.Sprintf(`{"x":"%s","y":"%s"}`, x, y)
	toRegisterValidation.Header.Set("captchaAnswer", captchaJSON)

	toRegisterValidation.Header.Set("email", "shayeganrood85@gmail.com")

	toRegisterValidation.Header.Set("csrf-token", csrf_token)
	client2 := &http.Client{}
	toRegisterValidationResponse, _ := client2.Do(toRegisterValidation)
	l("/login/validation POST status", toRegisterValidationResponse.Status)
	registerValidationJWTCookie := toRegisterValidationResponse.Cookies()[0]

	// /register/validation/jwt

	respak2, _ := http.Get("http://127.0.0.1:1234/token")
	bufjerk2, _ := io.ReadAll(respak2.Body)
	respak2.Body.Close()

	// GET
	bucket3, _ := http.NewRequest("GET", "http://127.0.0.1:1235/login/validation/jwt", nil)
	client000 := &http.Client{}
	bucket3.Header.Set("sideline", string(bufjerk2))
	respak000, _ := client000.Do(bucket3)
	bucket3Cookie := respak000.Cookies()[0]
	vals = strings.Split(bucket3Cookie.Value, ",")
	csrf_token = vals[1]
	l("/login/validation/jwt GET status", respak000.Status)

	// POST
	toRegisterValidationJWT, _ := http.NewRequest("POST", "http://127.0.0.1:1235/login/validation/jwt", nil)
	toRegisterValidationJWT.AddCookie(bucket3Cookie)
	toRegisterValidationJWT.AddCookie(registerValidationJWTCookie)
	toRegisterValidationJWT.Header.Set("csrf-token", csrf_token)
	toRegisterValidationJWT.Header.Set("password", "Lego138$$(oqwe")

	client3 := &http.Client{}
	toRegisterValidationJWTResponse, _ := client3.Do(toRegisterValidationJWT)

	registerValidationSubmitCookie := toRegisterValidationJWTResponse.Cookies()[0]
	l("/login/validation/jwt POST status", toRegisterValidationJWTResponse.Status)

	// /register/validation/submit

	var verificationCode string

	fmt.Println("Enter verification code:")
	fmt.Scan(&verificationCode)

	respak3, _ := http.Get("http://127.0.0.1:1234/token")

	bufjerk3, _ := io.ReadAll(respak3.Body)
	respak3.Body.Close()

	// GET
	bucket4, _ := http.NewRequest("GET", "http://127.0.0.1:1235/login/validation/submit", nil)
	client0000 := &http.Client{}
	bucket4.Header.Set("sideline", string(bufjerk3))
	respak0000, _ := client0000.Do(bucket4)
	bucket4Cookie := respak0000.Cookies()[0]
	vals = strings.Split(bucket4Cookie.Value, ",")
	csrf_token = vals[1]
	l("/login/validation/submit GET status", respak0000.Status)

	// POST
	toRegisterValidationSubmit, _ := http.NewRequest("POST", "http://127.0.0.1:1235/login/validation/submit", nil)
	toRegisterValidationSubmit.AddCookie(bucket4Cookie)
	toRegisterValidationSubmit.AddCookie(registerValidationSubmitCookie)
	toRegisterValidationSubmit.Header.Set("csrf-token", csrf_token)
	toRegisterValidationSubmit.Header.Set("verification", verificationCode)
	client4 := &http.Client{}
	toRegisterValidationSubmitResponse, _ := client4.Do(toRegisterValidationSubmit)
	fmt.Println("Final result:\n", toRegisterValidationSubmitResponse.Cookies()[0])
}
