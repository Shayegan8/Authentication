# Simple Authentication in Golang
This is actually for my website im currently working on it
It has captcha and verification with smtp server
Idea behind this is you have a reverse proxy that can capture client's ip addresses and give them to api endpoints in header of made up request
It uses Bucket limit system, give users a refresh token after passing all levels and with that refresh token they have access to other api endpoints and other
api endpoints in further development will give them a jwt token with this refresh token. refresh token will be changed after each login and it will have a timestamp. 

TODO:
- Add rest of the APIs
- Supporting OAuth2 too
- Delete account logic
