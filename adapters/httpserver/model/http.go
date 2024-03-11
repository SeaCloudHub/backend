package model

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
} // @name model.SuccessResponse

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Info    string `json:"info"`
} // @name model.ErrorResponse
