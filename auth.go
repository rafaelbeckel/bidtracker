package main

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

// JWTSecret to sign our tokens
const JWTSecret = "Dont-Tell-Mama!"

// Login issues a JWT to use in protected routes
// No password checking for simplicity
func Login(c echo.Context) error {
	username := c.FormValue("username")

	// username should be given
	if username == "" {
		return echo.ErrUnauthorized
	}

	// Create token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = username
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	// Generate encoded token
	tokenString, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		return err
	}

	// send token as response
	return c.JSON(http.StatusOK, map[string]string{
		"token": tokenString,
	})
}
