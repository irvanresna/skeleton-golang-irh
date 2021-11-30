package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"hypefast-api/lib/utils"

	validator "github.com/go-playground/validator/v10"
)

const (
	// XSignature custom header to hold signature string
	XSignature = "X-SIGNATURE"

	// XTimestamp custom header to hold timestamp that used for signature
	XTimestamp = "X-TIMESTAMPT"

	// XPlayer is a token that we get from OneSignal Push notification
	XPlayer = "X-PLAYER"

	// MsgSuccess ...
	MsgSuccess = "APP:SUCCESS"

	// MsgErrValidation ...
	MsgErrValidation = "ERR:VALIDATION"

	// MsgEmptyData Data not found ...
	MsgEmptyData = "ERR:EMPTY_DATA"

	// MsgErrParam error parameter argument or anything in query string
	MsgErrParam = "ERR:INVALID_PARAM"

	// MsgBadReq for general bad request
	MsgBadReq = "ERR:BAD_REQUEST"

	// MsgNotfound for not found 404 page
	MsgNotfound = "ERR:NOT_FOUND"

	// MsgAuthErr ..
	MsgAuthErr = "ERR:AUTHENTICATION"

	MsgAuthorizedErr = "ERR:AUTHORIZED"

	// HTTPTravellerChannel ...
	HTTPTravellerChannel = "webtraveller"

	// HTTPCmsChannel ...
	HTTPCmsChannel = "webcms"

	// XChannelHeader custom header for determine what the channel is
	XChannelHeader = "X-CHANNEL"

	AuthHeader = "Authorization"

	// AuthBase64Error flag for error base64
	AuthBase64Error = "[base64:Invalid]"
)

// ErrorBase64 give error string of invalid base64
func (h *App) ErrorBase64() error {
	return fmt.Errorf(AuthBase64Error)
}

// Bind bind the API request payload (body) into request struct.
func (h *App) Bind(r *http.Request, input interface{}) error {
	err := json.NewDecoder(r.Body).Decode(&input)

	return err
}

// GetChannel ...
func (h *App) GetChannel(r *http.Request) string {
	return r.Header.Get(XChannelHeader)
}

// GetToken ...
func (h *App) GetToken(r *http.Request) string {
	return r.Header.Get(AuthHeader)
}

// EmptyJSONArr ...
func (h *App) EmptyJSONArr() []map[string]interface{} {
	return []map[string]interface{}{}
}

// SendSuccess send success into response with 200 http code.
func (h *App) SendSuccess(w http.ResponseWriter, payload interface{}, pagination interface{}) {
	if pagination == nil {
		pagination = h.EmptyJSONArr()
	}
	h.RespondWithJSON(w, 200, MsgSuccess, "Success", payload, pagination)
}

// SendBadRequest send bad request into response with 400 http code.
func (h *App) SendBadRequest(w http.ResponseWriter, message string) {
	h.RespondWithJSON(w, 400, MsgBadReq, message, h.EmptyJSONArr(), h.EmptyJSONArr())
}

// SendBadWithNilDataRequest send bad request into response with 400 http code.
func (h *App) SendBadWithNilDataRequest(w http.ResponseWriter, message string) {
	h.RespondWithJSON(w, 400, MsgBadReq, message, nil, h.EmptyJSONArr())
}

// SendNotfound send bad request into response with 400 http code.
func (h *App) SendNotfound(w http.ResponseWriter, message string) {
	h.RespondWithJSON(w, 404, MsgNotfound, message, h.EmptyJSONArr(), h.EmptyJSONArr())
}

// SendAuthError send bad request into response with 400 http code.
func (h *App) SendAuthError(w http.ResponseWriter, message string) {
	h.RespondWithJSON(w, 401, MsgAuthErr, message, h.EmptyJSONArr(), h.EmptyJSONArr())
}

// SendUnAuthorizedData send bad request into response with 400 http code.
func (h *App) SendUnAuthorizedData(w http.ResponseWriter) {
	h.RespondWithJSON(w, 401, MsgAuthorizedErr, "unauthorized data", h.EmptyJSONArr(), h.EmptyJSONArr())
}

// SendRequestValidationError Send validation error response to consumers.
func (h *App) SendRequestValidationError(w http.ResponseWriter, validationErrors validator.ValidationErrors) {
	errorResponse := map[string][]string{}
	errorTranslation := validationErrors.Translate(h.Validator.Translator)
	// fmt.Println(errorTranslation)
	// fmt.Println(validationErrors)
	for _, err := range validationErrors {
		errKey := utils.Underscore(err.StructField())
		errorResponse[errKey] = append(
			errorResponse[errKey],
			strings.Replace(errorTranslation[err.Namespace()], err.StructField(), "[]", -1),
		)
	}

	h.RespondWithJSON(w, 400, MsgErrValidation, "validation error", errorResponse, h.EmptyJSONArr())
}

// RespondWithJSON write json response format
func (h *App) RespondWithJSON(
	w http.ResponseWriter,
	httpCode int,
	statCode string,
	message string,
	payload interface{},
	pagination interface{},
) {
	respPayload := map[string]interface{}{
		"stat_code":  statCode,
		"stat_msg":   message,
		"pagination": pagination,
		"data":       payload,
	}

	response, _ := json.Marshal(respPayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	_, _ = w.Write(response)
}

// GetUserID ...
func (h *App) GetUserID(ctx context.Context) (int64, error) {
	usX := fmt.Sprintf("%v", ctx.Value("user_id"))
	userID, err := strconv.ParseInt(usX, 10, 64)

	return userID, err
}

// GetUserEmail ...
func (h *App) GetUserEmail(ctx context.Context) string {
	emX := fmt.Sprintf("%v", ctx.Value("user_email"))
	return emX
}

// GetUserRole ...
func (h *App) GetUserRole(ctx context.Context) string {
	rolX := fmt.Sprintf("%v", ctx.Value("user_role_slug"))
	return rolX
}

// GetUserCategory ...
func (h *App) GetUserCategory(ctx context.Context) string {
	catX := fmt.Sprintf("%v", ctx.Value("user_role_category"))
	return catX
}

// ParamOrder ...
type ParamOrder struct {
	Field string
	By    string
}

// GetParamOrder Parse the url param to get order field & order by value
func (h *App) GetParamOrder(r *http.Request) (ParamOrder, error) {
	param := strings.Split(r.URL.Query().Get("order"), ",")
	pOrder := ParamOrder{}
	if len(param) != 2 {
		return pOrder, errors.New("wrong order parameters")
	}

	pOrder.Field = param[0]
	pOrder.By = param[1]

	return pOrder, nil
}

// GetIntParam Parse the url param to get value as integer.
// for example, we need to get limit and offset param
func (h *App) GetIntParam(r *http.Request, name string) (int, error) {
	param := r.URL.Query().Get(name)
	if len(param) == 0 {
		return 0, nil
	}

	return strconv.Atoi(param)
}

// GetStringParam Parse the url param to get value as string.
func (h *App) GetStringParam(r *http.Request, name string) (string, error) {
	param := r.URL.Query().Get(name)
	if len(param) == 0 {
		return "", nil
	}

	return param, nil
}

// GetBoolParam Parse the url param to get value as boolean
func (h *App) GetBoolParam(r *http.Request, name string) (bool, error) {
	param := r.URL.Query().Get(name)
	if len(param) == 0 {
		return false, nil
	}

	result, err := strconv.ParseBool(param)
	if err != nil {
		return false, err
	}

	return result, nil
}

func (h *App) PingAction(w http.ResponseWriter, r *http.Request) {
	h.SendSuccess(w, h.EmptyJSONArr(), nil)
}

// func generateSig(s reflect.Value, timestamp string) string {
// 	typeOfT := s.Type()
// 	combine := ""
// 	for i := 0; i < s.NumField(); i++ {
// 		f := s.Field(i)
// 		value := fmt.Sprintf("%v", f.Interface())
// 		vb := []byte(value)
// 		bs := sha1.Sum(vb)
// 		hVal := hex.EncodeToString(bs[:])

// 		combine += fmt.Sprintf("%s%s", typeOfT.Field(i).Tag.Get("json"), hVal)
// 	}
// 	// create complete sha1
// 	bSum := sha1.Sum([]byte(combine + timestamp))
// 	bSumVal := hex.EncodeToString(bSum[:])

// 	return bSumVal
// }

// // isValidSignature ...
// func isValidSignature(obj reflect.Value, timestamp, comparator string) bool {
// 	return generateSig(obj, timestamp) == comparator
// }

// // isValidSettingSignature ...
// func isValidSettingSignature(r *http.Request, key string) bool {
// 	sig := r.Header.Get(XSignature)
// 	ts := r.Header.Get(XTimestamp)

// 	bSum := sha1.Sum([]byte(key + ts))
// 	bSumVal := hex.EncodeToString(bSum[:])

// 	return sig == bSumVal
// }
