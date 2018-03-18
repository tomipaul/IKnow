package controller

import (
	. "WeKnow_api/pgModel"
	utils "WeKnow_api/utilities"
	"encoding/json"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
)

func CreateCollectionEndPoint(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	collection := &Collection{}

	if err := json.NewDecoder(r.Body).Decode(collection); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	} else {
		decodedClaims := context.Get(r, "decoded")
		userId := decodedClaims.(jwt.MapClaims)["userId"].(float64)
		if err := utils.ValidateNewCollection(*collection); err == nil {
			collection.UserId = int64(userId)
			if err := db.Insert(collection); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}

			utils.RespondWithSuccess(w, http.StatusCreated, fmt.Sprintf("%s collection was created successfully", string(collection.Name)), "message")
		} else {
			utils.RespondWithJsonError(w, http.StatusBadRequest, err.Error())
		}

	}
}
