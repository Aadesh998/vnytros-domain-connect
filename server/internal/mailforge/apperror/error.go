package apperror

var ErrUnauthorized = AppError{
	Code:       "UNAUTHORIZED",
	Message:    "Unauthorized access",
	HTTPStatus: 401,
}

var TokenExpired = AppError{
	Code:       "TOKEN_EXPIRED",
	Message:    "Token has expired",
	HTTPStatus: 401,
}

var BadRequest = AppError{
	Code:       "BAD_REQUEST",
	Message:    "Bad request",
	HTTPStatus: 400,
}

var NotFound = AppError{
	Code:       "NOT_FOUND",
	Message:    "Resource not found",
	HTTPStatus: 404,
}

var Conflict = AppError{
	Code:       "CONFLICT",
	Message:    "Resource already exists",
	HTTPStatus: 409,
}

var SmtpNotConfigured = AppError{
	Code:       "SMTP_NOT_CONFIGURED",
	Message:    "Email credentials are not configured. Add a sender in Settings before sending.",
	HTTPStatus: 400,
}

var CampaignNotSendable = AppError{
	Code:       "CAMPAIGN_NOT_SENDABLE",
	Message:    "This campaign has already been dispatched",
	HTTPStatus: 409,
}

var NoRecipients = AppError{
	Code:       "NO_RECIPIENTS",
	Message:    "The uploaded file contained no valid email addresses",
	HTTPStatus: 400,
}

var QueueUnavailable = AppError{
	Code:       "QUEUE_UNAVAILABLE",
	Message:    "The sending queue is unavailable, please retry shortly",
	HTTPStatus: 503,
}

var InternalServerError = AppError{
	Code:       "INTERNAL_SERVER_ERROR",
	Message:    "Internal server error",
	HTTPStatus: 500,
}
