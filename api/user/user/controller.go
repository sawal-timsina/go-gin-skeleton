package user

import (
	"boilerplate-api/internal/api_errors"
	"boilerplate-api/internal/config"
	"boilerplate-api/internal/constants"
	"boilerplate-api/internal/json_response"
	"boilerplate-api/internal/request_validator"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	logger      config.Logger
	userService Service
	env         config.Env
	validator   request_validator.Validator
}

// NewController Creates New user controller
func NewController(
	logger config.Logger,
	userService Service,
	env config.Env,
	validator request_validator.Validator,
) Controller {
	return Controller{
		logger:      logger,
		userService: userService,
		env:         env,
		validator:   validator,
	}
}

// @Tags			UserApi
// @Summary		User Profile
// @Description	get user profile
// @Security		Bearer
// @Produce		application/json
// @Success		200	{object}	json_response.Data[CUser]
// @Failure		500	{object}	api_errors.Envelope
// @Router			/api/v1/profile [get]
// @Id				GetUserProfile
func (cc Controller) GetUserProfile(c *gin.Context) {
	userID := fmt.Sprintf("%v", c.MustGet(constants.UserID))

	user, err := cc.userService.GetOneUser(userID)
	if err != nil {
		api_errors.RespondError(c, api_errors.Wrap(err, http.StatusInternalServerError, api_errors.CodeInternal, "Failed to get user's profile data"))
		return
	}

	c.JSON(
		http.StatusOK, json_response.Data[CUser]{
			Data: user,
		},
	)
}
