package main_test

import (
	main "WeKnow_api"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/haoxins/supertest"
	"github.com/subosito/gotenv"

	. "WeKnow_api/model"

	. "WeKnow_api/handler"
)

var h *Handler

var app main.App

func TestMain(m *testing.M) {
	gotenv.Load()
	dbConfig := map[string]string{
		"User":     os.Getenv("TEST_DB_USERNAME"),
		"Password": os.Getenv("TEST_DB_PASSWORD"),
		"Database": os.Getenv("TEST_DATABASE"),
	}
	app = main.CreateApp(dbConfig)
	os.Exit(m.Run())
}

func TestUserProfile(t *testing.T) {

	testServer := httptest.NewServer(app.Router)
	defer testServer.Close()

	user := User{
		Username:    "test",
		Email:       "test@gmail.com",
		PhoneNumber: "08123425634",
		Password:    "test",
	}

	err := app.Db.Insert(&user)

	if err != nil {
		t.Log(err.Error())
	}

	token, _ := user.GenerateToken()
	userToken := "Bearer " + token

	t.Run("cannot update with no username", func(t *testing.T) {
		Request(testServer.URL, t).
			Put("/api/v1/user/profile").
			Set("authorization", userToken).
			Send(`{"username": ""}`).
			Expect(400).
			Expect("Content-Type", "application/json").
			Expect(`{"error":"Username cannot be empty"}`).
			End()
	})

	t.Run("cannot update with no phone number", func(t *testing.T) {
		Request(testServer.URL, t).
			Put("/api/v1/user/profile").
			Set("authorization", userToken).
			Send(`{"phoneNumber": ""}`).
			Expect(400).
			Expect("Content-Type", "application/json").
			Expect(`{"error":"Phone number cannot be empty"}`).
			End()
	})

	t.Run("cannot update with empty email", func(t *testing.T) {
		Request(testServer.URL, t).
			Put("/api/v1/user/profile").
			Set("authorization", userToken).
			Send(`{"email": "" }`).
			Expect(400).
			Expect("Content-Type", "application/json").
			Expect(`{"error": "Enter a valid email"}`).
			End()
	})

	t.Run("cannot update without valid email", func(t *testing.T) {
		Request(testServer.URL, t).
			Put("/api/v1/user/profile").
			Set("authorization", userToken).
			Send(`{"email": "testemail" }`).
			Expect(400).
			Expect("Content-Type", "application/json").
			Expect(`{"error": "Enter a valid email"}`).
			End()
	})

	t.Run("updates with valid username", func(t *testing.T) {
		Request(testServer.URL, t).
			Put("/api/v1/user/profile").
			Set("authorization", userToken).
			Send(`{"username": "tester"}`).
			Expect(200).
			Expect("Content-Type", "application/json").
			Expect(`{"message":"ProfileUpdatedsuccessfully","updatedProfile":{"username":"tester"}}`).
			End()
	})

	t.Run("updates with valid phone number", func(t *testing.T) {
		Request(testServer.URL, t).
			Put("/api/v1/user/profile").
			Set("authorization", userToken).
			Send(`{"phoneNumber": "09023450022" }`).
			Expect(200).
			Expect("Content-Type", "application/json").
			Expect(`{"message":"ProfileUpdatedsuccessfully","updatedProfile":{"phoneNumber":"09023450022"}}`).
			End()
	})

	if err := app.Db.Delete(&user); err != nil {
		t.Log(err.Error())
	}

}
