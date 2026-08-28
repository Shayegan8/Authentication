package main

import (
	"fmt"
	"net/http"
	"strings"
)

var l = fmt.Println

func main() {

	// /register GET,POST

	bucket1, _ := http.Get("http://127.0.0.1:1234/register")

	// GET
	bucket1Cookie := bucket1.Cookies()[0]
	vals := strings.Split(bucket1Cookie.Value, ",")
	csrf_token := vals[1]
	l("/register GET status", bucket1.Status)
	bucket1.Body.Close()

	// POST
	toRegister, _ := http.NewRequest("POST", "http://127.0.0.1:1234/register", nil)
	toRegister.AddCookie(bucket1Cookie)

	toRegister.Header.Set("csrf-token", csrf_token)
	client := &http.Client{}
	toRegisterResponse, _ := client.Do(toRegister)
	l("/register POST status", toRegisterResponse.Status)

	captchaGeneratedCookie := toRegisterResponse.Cookies()[0]
	// /register/validation GET,POST

	// GET
	bucket2, _ := http.Get("http://127.0.0.1:1234/register/validation")
	bucket2Cookie := bucket2.Cookies()[0]
	vals = strings.Split(bucket2Cookie.Value, ",")
	csrf_token = vals[1]
	l("/register/validation GET status", bucket2.Status)

	bucket2.Body.Close()

	var x string
	var y string
	fmt.Println("Please enter the result with order of first x and y: ")
	fmt.Scan(&x, &y)

	// POST

	toRegisterValidation, _ := http.NewRequest("POST", "http://127.0.0.1:1234/register/validation", nil)
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
	l("/register/validation POST status", toRegisterValidationResponse.Status)
	registerValidationJWTCookie := toRegisterValidationResponse.Cookies()[0]

	// /register/validation/jwt

	// GET
	bucket3, _ := http.Get("http://127.0.0.1:1234/register/validation/jwt")
	bucket3Cookie := bucket3.Cookies()[0]
	vals = strings.Split(bucket3Cookie.Value, ",")
	csrf_token = vals[1]
	l("/register/validation/jwt GET status", bucket3.Status)

	bucket3.Body.Close()

	// POST
	toRegisterValidationJWT, _ := http.NewRequest("POST", "http://127.0.0.1:1234/register/validation/jwt", nil)
	toRegisterValidationJWT.AddCookie(bucket3Cookie)
	toRegisterValidationJWT.AddCookie(registerValidationJWTCookie)
	toRegisterValidationJWT.Header.Set("csrf-token", csrf_token)
	toRegisterValidationJWT.Header.Set("username", "Shayegan8")
	toRegisterValidationJWT.Header.Set("password", "Lego138$$(oqwe")

	client3 := &http.Client{}
	toRegisterValidationJWTResponse, _ := client3.Do(toRegisterValidationJWT)

	registerValidationSubmitCookie := toRegisterValidationJWTResponse.Cookies()[0]
	l("/register/validation/jwt POST status", toRegisterValidationJWTResponse.Status)

	// /register/validation/submit

	var verificationCode string

	fmt.Println("Enter verification code:")
	fmt.Scan(&verificationCode)

	// GET
	bucket4, _ := http.Get("http://127.0.0.1:1234/register/validation/submit")
	bucket4Cookie := bucket4.Cookies()[0]
	vals = strings.Split(bucket4Cookie.Value, ",")
	csrf_token = vals[1]
	l("/register/validation/jwt GET status", bucket4.Status)

	bucket4.Body.Close()

	// POST
	toRegisterValidationSubmit, _ := http.NewRequest("POST", "http://127.0.0.1:1234/register/validation/submit", nil)
	toRegisterValidationSubmit.AddCookie(bucket4Cookie)
	toRegisterValidationSubmit.AddCookie(registerValidationSubmitCookie)
	toRegisterValidationSubmit.Header.Set("csrf-token", csrf_token)
	toRegisterValidationSubmit.Header.Set("verification", verificationCode)
	client4 := &http.Client{}
	toRegisterValidationSubmitResponse, _ := client4.Do(toRegisterValidationSubmit)
	fmt.Println("Final result:\n", toRegisterValidationSubmitResponse.Cookies()[0])
}
