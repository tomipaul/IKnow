package handler

import (
	. "WeKnow_api/model"
	utils "WeKnow_api/utilities"
	"encoding/json"
	"net/http"
	"strconv"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-pg/pg"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

// PostResource post a new resource
func (h *Handler) PostResource(w http.ResponseWriter, r *http.Request) {
	var resource struct {
		Resource
		Tags []string
	}
	err := json.NewDecoder(r.Body).Decode(&resource)
	if err != nil {
		utils.RespondWithError(
			w, http.StatusBadRequest,
			"Invalid resource field(s) in request payload",
		)
	} else {
		decodedClaims := context.Get(r, "decoded")
		userId := decodedClaims.(jwt.MapClaims)["userId"].(float64)
		resource.UserId = int64(userId)
		err := utils.ValidateNewResource(&resource.Resource)
		if err != nil {
			utils.RespondWithJsonError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}
		if err := h.Db.Insert(&resource.Resource); err != nil {
			if err.(pg.Error).Field('C') == "23505" {
				utils.RespondWithError(
					w, http.StatusConflict,
					"A resource exists with provided link",
				)
			} else {
				utils.RespondWithError(
					w,
					http.StatusInternalServerError,
					"Something went wrong",
				)
			}
		} else {
			if tags, Ok := context.GetOk(r, "tags"); Ok {
				var resourceTags []interface{}
				for _, tag := range tags.([]interface{}) {
					resourceTags = append(resourceTags, &ResourceTag{
						TagId:      tag.(*Tag).Id,
						ResourceId: resource.Id,
					})
				}
				if err := h.Db.Insert(resourceTags...); err != nil {
					utils.RespondWithError(
						w, http.StatusInternalServerError,
						"Oops! we couldn't attach tags to the resource",
					)
					return
				}
			}
			payload := map[string]interface{}{
				"resource": resource.Resource,
				"tags":     resource.Tags,
				"message":  "Resource created",
			}
			utils.RespondWithJson(w, http.StatusCreated, payload)
		}
	}
	return
}

// UpdateResource update an existing resource
func (h *Handler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload map[string]interface{}
	json.NewDecoder(r.Body).Decode(&payload)
	if err := utils.ValidateUpdateResourcePayload(payload); err != nil {
		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}
	userId := context.Get(r, "decoded").(jwt.MapClaims)["userId"].(float64)
	resourceId, _ := strconv.ParseInt(mux.Vars(r)["resourceId"], 10, 64)
	if err := utils.ValidateResourceId(resourceId); err != nil {
		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}
	resource := &Resource{Id: resourceId, UserId: int64(userId)}
	updatedFields := []string{}
	for key, value := range payload {
		switch key {
		case "title":
			resource.Title = value.(string)
			updatedFields = append(updatedFields, "title")
		case "link":
			resource.Link = value.(string)
			updatedFields = append(updatedFields, "link")
		case "type":
			resource.Type = value.(string)
			updatedFields = append(updatedFields, "type")
		case "privacy":
			resource.Privacy = value.(string)
			updatedFields = append(updatedFields, "privacy")
		}
	}
	if len(updatedFields) > 0 {
		_, err := h.Db.
			Model(resource).
			Column(updatedFields...).
			Where("id = ?id AND user_id = ?user_id").
			Returning("*").
			Update(resource)

		if err != nil {
			if err == pg.ErrNoRows {
				utils.RespondWithError(
					w, http.StatusNotFound,
					"Either this resource does not exist or you cannot access it",
				)
			} else if pgError, OK := err.(pg.Error); OK && pgError.Field('C') == "23505" {
				utils.RespondWithError(
					w, http.StatusConflict,
					"A resource exists with provided link",
				)
			} else {
				utils.RespondWithError(
					w, http.StatusInternalServerError, "Something went wrong",
				)
			}
			return
		}
	}
	tags, Ok := context.GetOk(r, "tags")
	var addedTagTitles []string
	if Ok {
		var resourceTags []interface{}
		for _, tag := range tags.([]interface{}) {
			resourceTags = append(resourceTags, &ResourceTag{
				TagId:      tag.(*Tag).Id,
				ResourceId: resource.Id,
			})
			addedTagTitles = append(addedTagTitles, tag.(*Tag).Title)
		}
		if err := h.Db.Insert(resourceTags...); err != nil {
			utils.RespondWithError(
				w, http.StatusInternalServerError,
				"Oops! we couldn't attach added tags to the resource",
			)
			return
		}
	}
	removedTags, Ok := context.GetOk(r, "removed_tags")
	var removedTagTitles []string
	if Ok {
		var tagIds []int64
		for _, tag := range removedTags.([]interface{}) {
			tagIds = append(tagIds, tag.(*Tag).Id)
			removedTagTitles = append(removedTagTitles, tag.(*Tag).Title)
		}
		if _, err := h.Db.Model(&ResourceTag{}).
			Where("tag_id in (?)", pg.In(tagIds)).
			Delete(); err != nil {
			utils.RespondWithError(
				w, http.StatusInternalServerError,
				"Oops! we couldn't detach removed tags from the resource",
			)
			return
		}
	}
	responsePayload := map[string]interface{}{
		"updatedResource": resource,
		"addedTags":       addedTagTitles,
		"removedTags":     removedTagTitles,
		"message":         "resource updated successfully",
	}
	utils.RespondWithJson(w, http.StatusOK, responsePayload)
	return
}

// DeleteResource delete an existing resource
func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	resourceId, _ := strconv.ParseInt(mux.Vars(r)["resourceId"], 10, 64)
	if resourceId == 0 {
		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			"Invalid resource Id in request",
		)
		return
	}
	userId := context.Get(r, "decoded").(jwt.MapClaims)["userId"].(float64)
	resource := Resource{Id: resourceId, UserId: int64(userId)}
	res, err := h.Db.
		Model(&resource).
		Where("id = ?id AND user_id = ?user_id").
		Delete()
	if res.RowsAffected() == 0 {
		utils.RespondWithError(
			w,
			http.StatusForbidden,
			"Either this resource does not exist or you cannot access it",
		)
		return
	}
	if err != nil {
		utils.RespondWithError(
			w,
			http.StatusInternalServerError,
			"Something went wrong",
		)
	} else {
		payload := map[string]interface{}{
			"message":         "Resource deleted successfully",
			"deletedResource": resourceId,
		}
		utils.RespondWithJson(w, http.StatusOK, payload)
	}
	return
}

// RecommendResource recommend a resource
func (h *Handler) RecommendResource(w http.ResponseWriter, r *http.Request) {
	resourceId, _ := strconv.ParseInt(mux.Vars(r)["resourceId"], 10, 64)
	if err := utils.ValidateResourceId(resourceId); err != nil {
		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	userId := context.Get(r, "decoded").(jwt.MapClaims)["userId"].(float64)
	var recommendationCount int64

	err := h.Db.RunInTransaction(func(tx *pg.Tx) error {
		_, err := tx.QueryOne(
			pg.Scan(&recommendationCount),
			`SELECT recommendations FROM resources WHERE id = ? FOR UPDATE`,
			resourceId,
		)
		if err != nil {
			return err
		}

		recommendationCount++
		_, err = tx.Exec(
			`INSERT INTO recommendations (user_id, resource_id)
			VALUES (?0, ?1);
			UPDATE resources SET recommendations = ?2 WHERE id = ?1`,
			int64(userId), resourceId, recommendationCount,
		)
		return err
	})
	if err != nil {
		if err == pg.ErrNoRows {
			utils.RespondWithError(
				w, http.StatusNotFound, "Resource does not exist",
			)
		} else if pgError, OK := err.(pg.Error); OK && pgError.Field('C') == "23505" {
			utils.RespondWithError(
				w, http.StatusConflict,
				"You have recommended this resource",
			)
		} else {
			utils.RespondWithError(
				w, http.StatusInternalServerError, "Something went wrong",
			)
		}
	} else {
		payload := map[string]interface{}{
			"message":             "Recommend resource successful",
			"recommendationCount": recommendationCount,
		}
		utils.RespondWithJson(w, http.StatusOK, payload)
	}
	return
}

// GetResource get a resource
func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	resourceId, _ := strconv.ParseInt(mux.Vars(r)["resourceId"], 10, 64)
	if err := utils.ValidateResourceId(resourceId); err != nil {
		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}
	userId := context.Get(r, "decoded").(jwt.MapClaims)["userId"].(float64)
	resource := Resource{Id: resourceId}

	condition := `resource.id = ?0 AND
	(resource.privacy = 'public' OR
	resource.user_id = ?1 OR
	(resource.privacy = 'followers' AND
		(EXISTS(SELECT * FROM connections WHERE initiator_id = ?1 AND
			recipient_id = resource.user_id))))
	`
	err := h.Db.
		Model(&resource).
		Column("resource.*", "Tags").
		ColumnExpr(
			`a_user.username AS user__username,
			a_user.email AS user__email`,
		).
		Join("JOIN users AS a_user ON a_user.id = resource.user_id").
		Where(condition, resource.Id, int64(userId)).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			utils.RespondWithError(
				w,
				http.StatusNotFound,
				"Either this resource does not exist or you cannot access it",
			)
			return
		}
		utils.RespondWithError(
			w,
			http.StatusInternalServerError,
			"Something went wrong",
		)
	} else {
		payload := map[string]interface{}{
			"resource": resource,
		}
		utils.RespondWithJson(w, http.StatusOK, payload)
	}
	return
}
