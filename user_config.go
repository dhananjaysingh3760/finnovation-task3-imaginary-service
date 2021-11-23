package main

import (
	// "encoding/json"
	// "fmt"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("sEcRetkEy")

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func GetFirebaseConfig(r *http.Request) (c *Credentials, err error) {
	tokenStr := r.Header.Get("Authorization")
	tokenStr = tokenStr[7:]
	if err != nil {
		if err == http.ErrNoCookie {
			return
		}
		return
	}
	claims := &Claims{
		Username: userId,
	}

	tkn, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return
		}
		return
	}

	if !tkn.Valid {
		return
	}
	fmt.Println(userId)

	c, err1 := GetCredentials(userId)
	fmt.Println(c)
	if err1 != nil {
		fmt.Println(err1)
	}
	return c, nil
}

func register(c string, b string, f string, w http.ResponseWriter, r *http.Request) {
	cred := &Credentials{
		UserId:                    userId,
		ClientResourceStorageName: c,
		BucketName:                b,
		FolderPath:                f,
	}
	test, err := Save(cred)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(test)
	claims := &Claims{
		Username: userId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var bearer = "Bearer " + tokenString
	w.Header().Add("Authorization", bearer)
}

// gs://chat-app-5b482.appspot.com

// http.SetCookie(w,
// 	&http.Cookie{
// 		Name:  "token",
// 		Value: tokenString,
// 	})
