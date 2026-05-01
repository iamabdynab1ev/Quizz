package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"request-system/config"
	"request-system/internal/dto"
	"request-system/internal/services"
	"request-system/pkg/contextkeys"
	apperrors "request-system/pkg/errors"
	"request-system/pkg/filestorage"
	"request-system/pkg/service"
	"request-system/pkg/utils"
	"request-system/pkg/validation"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type AuthController struct {
	authService           services.AuthServiceInterface
	authPermissionService services.AuthPermissionServiceInterface
	jwtSvc                service.JWTService
	fileStorage           filestorage.FileStorageInterface
	logger                *zap.Logger
}

func NewAuthController(
	authService services.AuthServiceInterface,
	authPermissionService services.AuthPermissionServiceInterface,
	jwtSvc service.JWTService,
	fileStorage filestorage.FileStorageInterface,
	logger *zap.Logger,
) *AuthController {
	return &AuthController{
		authService:           authService,
		authPermissionService: authPermissionService,
		jwtSvc:                jwtSvc,
		fileStorage:           fileStorage,
		logger:                logger,
	}
}

func (ctrl *AuthController) errorResponse(c echo.Context, err error) error {
	return utils.ErrorResponse(c, err, ctrl.logger)
}

func (ctrl *AuthController) Login(c echo.Context) error {
	var payload dto.LoginDTO

	if err := c.Bind(&payload); err != nil {
		ctrl.logger.Error("Login: РѕС€РёР±РєР° РїСЂРёРІСЏР·РєРё РґР°РЅРЅС‹С…", zap.Error(err))
		return ctrl.errorResponse(c, apperrors.NewBadRequestError("РќРµРІРµСЂРЅС‹Р№ С„РѕСЂРјР°С‚ РґР°РЅРЅС‹С… РґР»СЏ РІС…РѕРґР°"))
	}

	if err := c.Validate(&payload); err != nil {
		ctrl.logger.Error("Login: РѕС€РёР±РєР° РІР°Р»РёРґР°С†РёРё РґР°РЅРЅС‹С…", zap.Error(err))
		return ctrl.errorResponse(c, err)
	}

	user, err := ctrl.authService.Login(c.Request().Context(), payload)
	if err != nil {
		ctrl.logger.Error("Login: РѕС€РёР±РєР° Р°РІС‚РѕСЂРёР·Р°С†РёРё", zap.String("login", payload.Login), zap.Error(err))
		return ctrl.errorResponse(c, err)
	}

	permissions, err := ctrl.authPermissionService.GetAllUserPermissions(c.Request().Context(), user.ID)
	if err != nil {
		ctrl.logger.Error("Login: РЅРµ СѓРґР°Р»РѕСЃСЊ РїРѕР»СѓС‡РёС‚СЊ РїСЂРёРІРёР»РµРіРёРё РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ", zap.Uint64("userID", user.ID), zap.Error(err))
		permissions = []string{}
	}

	return ctrl.generateTokensAndRespond(c, user.ID, permissions, "РђРІС‚РѕСЂРёР·Р°С†РёСЏ РїСЂРѕС€Р»Р° СѓСЃРїРµС€РЅРѕ", payload.RememberMe, "")
}
func (ctrl *AuthController) Logout(c echo.Context) error {
	if cookie, err := c.Cookie("refreshToken"); err == nil && cookie != nil && cookie.Value != "" {
		if _, sessionID, err := ctrl.jwtSvc.ValidateRefreshToken(cookie.Value); err == nil {
			if revokeErr := ctrl.authService.InvalidateRefreshSession(c.Request().Context(), sessionID); revokeErr != nil {
				ctrl.logger.Warn("Logout: РЅРµ СѓРґР°Р»РѕСЃСЊ РѕС‚РѕР·РІР°С‚СЊ refresh session", zap.String("sessionID", sessionID), zap.Error(revokeErr))
			}
		}
	}

	cookie := &http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	c.SetCookie(cookie)

	return utils.SuccessResponse(c, nil, "Р’С‹ СѓСЃРїРµС€РЅРѕ РІС‹С€Р»Рё РёР· СЃРёСЃС‚РµРјС‹.", http.StatusOK)
}
func (ctrl *AuthController) RefreshToken(c echo.Context) error {
	cookie, err := c.Cookie("refreshToken")
	if err != nil {
		return utils.ErrorResponse(c, apperrors.ErrUnauthorized, ctrl.logger)
	}
	refreshTokenString := cookie.Value

	userID, sessionID, err := ctrl.jwtSvc.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return utils.ErrorResponse(c, err, ctrl.logger)
	}

	if err := ctrl.authService.ValidateRefreshSession(c.Request().Context(), userID, sessionID); err != nil {
		ctrl.logger.Warn("RefreshToken: РЅРµР°РєС‚РёРІРЅР°СЏ РёР»Рё РѕС‚РѕР·РІР°РЅРЅР°СЏ refresh session", zap.Uint64("userID", userID), zap.String("sessionID", sessionID), zap.Error(err))
		return utils.ErrorResponse(c, apperrors.ErrUnauthorized, ctrl.logger)
	}

	permissions, err := ctrl.authPermissionService.GetAllUserPermissions(c.Request().Context(), userID)
	if err != nil {
		ctrl.logger.Error("РќРµ СѓРґР°Р»РѕСЃСЊ РїРѕР»СѓС‡РёС‚СЊ РїСЂРёРІРёР»РµРіРёРё РїСЂРё РѕР±РЅРѕРІР»РµРЅРёРё С‚РѕРєРµРЅР°", zap.Uint64("userID", userID), zap.Error(err))
		permissions = []string{}
	}

	return ctrl.generateTokensAndRespond(
		c,
		userID,
		permissions,
		"РўРѕРєРµРЅС‹ СѓСЃРїРµС€РЅРѕ РѕР±РЅРѕРІР»РµРЅС‹",
		true,
		sessionID,
	)
}
func (ctrl *AuthController) Me(c echo.Context) error {
	userID, ok := c.Request().Context().Value(contextkeys.UserIDKey).(uint64)
	if !ok || userID == 0 {
		ctrl.logger.Error("Не удалось получить userID из контекста в защищенном маршруте")
		return utils.ErrorResponse(c, apperrors.ErrUnauthorized, ctrl.logger)
	}

	userProfile, err := ctrl.authService.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		ctrl.logger.Error("Ошибка получения профиля пользователя по ID", zap.Uint64("userID", userID), zap.Error(err))
		return utils.ErrorResponse(c, err, ctrl.logger)
	}

	return utils.SuccessResponse(c, userProfile, "Профиль пользователя успешно получен", http.StatusOK)
}

func (ctrl *AuthController) RequestPasswordReset(c echo.Context) error {
	var payload dto.ResetPasswordRequestDTO
	if err := c.Bind(&payload); err != nil {
		ctrl.logger.Error("Ошибка при привязке данных", zap.Error(err))
		return ctrl.errorResponse(c, err)
	}

	if err := c.Validate(&payload); err != nil {
		ctrl.logger.Error("Ошибка при валидации данных", zap.Any("payload", payload), zap.Error(err))
		return ctrl.errorResponse(c, err)
	}

	if err := ctrl.authService.RequestPasswordReset(c.Request().Context(), payload); err != nil {
		ctrl.logger.Error("Ошибка при запросе сброса пароля", zap.Any("payload", payload), zap.Error(err))
		return ctrl.errorResponse(c, err)
	}
	return utils.SuccessResponse(c, nil, "Если пользователь существует, инструкция будет отправлена.", http.StatusOK)
}

func (ctrl *AuthController) VerifyCode(c echo.Context) error {
	var payload dto.VerifyCodeDTO
	if err := c.Bind(&payload); err != nil {
		return ctrl.errorResponse(c, apperrors.ErrBadRequest)
	}
	if err := c.Validate(&payload); err != nil {
		return ctrl.errorResponse(c, err)
	}

	response, err := ctrl.authService.VerifyResetCode(c.Request().Context(), payload)
	if err != nil {
		return ctrl.errorResponse(c, err)
	}

	return utils.SuccessResponse(c, response, "Код подтвержден.", http.StatusOK)
}

func (ctrl *AuthController) ResetPassword(c echo.Context) error {
	var payload dto.ResetPasswordDTO
	if err := c.Bind(&payload); err != nil {
		return ctrl.errorResponse(c, apperrors.ErrBadRequest)
	}
	if err := c.Validate(&payload); err != nil {
		return ctrl.errorResponse(c, err)
	}

	if err := ctrl.authService.ResetPassword(c.Request().Context(), payload); err != nil {
		return ctrl.errorResponse(c, err)
	}
	return utils.SuccessResponse(c, nil, "Пароль успешно изменен.", http.StatusOK)
}
func (ctrl *AuthController) generateTokensAndRespond(c echo.Context, userID uint64, permissions []string, message string, rememberMe bool, revokeSessionID string) error {
	accessTokenTTL := ctrl.jwtSvc.GetAccessTokenTTL()
	var refreshTokenTTL time.Duration

	if rememberMe {
		refreshTokenTTL = ctrl.jwtSvc.GetRefreshTokenTTL()
	} else {
		refreshTokenTTL = time.Hour * 8
	}

	sessionID := uuid.NewString()
	ctx := c.Request().Context()
	if err := ctrl.authService.RegisterRefreshSession(ctx, userID, sessionID, refreshTokenTTL); err != nil {
		ctrl.logger.Error("РќРµ СѓРґР°Р»РѕСЃСЊ Р·Р°СЂРµРіРёСЃС‚СЂРёСЂРѕРІР°С‚СЊ refresh session", zap.Error(err), zap.Uint64("userID", userID), zap.String("sessionID", sessionID))
		return ctrl.errorResponse(c, err)
	}

	accessToken, refreshToken, err := ctrl.jwtSvc.GenerateTokens(userID, 0, sessionID, accessTokenTTL, refreshTokenTTL)
	if err != nil {
		if revokeErr := ctrl.authService.InvalidateRefreshSession(ctx, sessionID); revokeErr != nil {
			ctrl.logger.Warn("РќРµ СѓРґР°Р»РѕСЃСЊ РѕС‚РјРµРЅРёС‚СЊ freshly registered session after token generation error", zap.Error(revokeErr), zap.Uint64("userID", userID), zap.String("sessionID", sessionID))
		}
		ctrl.logger.Error("РќРµ СѓРґР°Р»РѕСЃСЊ СЃРіРµРЅРµСЂРёСЂРѕРІР°С‚СЊ С‚РѕРєРµРЅС‹", zap.Error(err), zap.Uint64("userID", userID))
		return ctrl.errorResponse(c, err)
	}

	if revokeSessionID != "" && revokeSessionID != sessionID {
		if err := ctrl.authService.InvalidateRefreshSession(ctx, revokeSessionID); err != nil {
			ctrl.logger.Warn("РќРµ СѓРґР°Р»РѕСЃСЊ РѕС‚РѕР·РІР°С‚СЊ previous refresh session", zap.Error(err), zap.Uint64("userID", userID), zap.String("sessionID", revokeSessionID))
		}
	}

	cookie := new(http.Cookie)
	cookie.Name = "refreshToken"
	cookie.Value = refreshToken
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.SameSite = http.SameSiteNoneMode

	if rememberMe {
		cookie.Expires = time.Now().Add(refreshTokenTTL)
	}

	c.SetCookie(cookie)

	response := dto.AuthResponseDTO{
		AccessToken: accessToken,
		Permissions: permissions,
	}

	return utils.SuccessResponse(c, response, message, http.StatusOK)
}
func (ctrl *AuthController) UpdateMe(c echo.Context) error {
	reqCtx := c.Request().Context()
	var payload dto.UpdateMyProfileDTO

	// 1. Р В§Р С‘РЎвЂљР В°Р ВµР С JSON Р Т‘Р В°Р Р…Р Р…РЎвЂ№Р Вµ
	dataString := c.FormValue("data")

	// Р В­РЎвЂљР С• Р С”Р В°РЎР‚РЎвЂљР В° Р Т‘Р В»РЎРЏ Р С•РЎвЂљРЎРѓР В»Р ВµР В¶Р С‘Р Р†Р В°Р Р…Р С‘РЎРЏ: РЎвЂЎРЎвЂљР С• Р С‘Р СР ВµР Р…Р Р…Р С• Р С—РЎР‚Р С‘РЎРѓР В»Р В°Р В» РЎвЂћРЎР‚Р С•Р Р…РЎвЂљР ВµР Р…Р Т‘?
	explicitFields := make(map[string]interface{})

	if dataString != "" {
		_ = json.Unmarshal([]byte(dataString), &payload)
		_ = json.Unmarshal([]byte(dataString), &explicitFields)
	}

	// 2. Р С›Р В±РЎР‚Р В°Р В±Р В°РЎвЂљРЎвЂ№Р Р†Р В°Р ВµР С Р В·Р В°Р С–РЎР‚РЎС“Р В·Р С”РЎС“ РЎвЂћР В°Р в„–Р В»Р В°
	photoURL, err := ctrl.handlePhotoUpload(c, "profile_photo")
	if err != nil {
		return ctrl.errorResponse(c, err)
	}

	if photoURL != nil {

		payload.PhotoURL = photoURL
	} else {

		if val, exists := explicitFields["photo_url"]; exists && val == nil {
			deleteSignal := "SET_NULL"
			payload.PhotoURL = &deleteSignal
		} else {

			payload.PhotoURL = nil
		}
	}

	updatedUser, err := ctrl.authService.UpdateMyProfile(reqCtx, payload)
	if err != nil {
		if photoURL != nil {
			_ = ctrl.fileStorage.Delete(*photoURL)
		}
		return ctrl.errorResponse(c, err)
	}

	return utils.SuccessResponse(c, updatedUser, "Р СџРЎР‚Р С•РЎвЂћР С‘Р В»РЎРЉ Р С•Р В±Р Р…Р С•Р Р†Р В»Р ВµР Р…", http.StatusOK)
}

func (c *AuthController) handlePhotoUpload(ctx echo.Context, uploadContext string) (*string, error) {
	file, err := ctx.FormFile("photoFile")
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, apperrors.NewHttpError(http.StatusBadRequest, "Р С›РЎв‚¬Р С‘Р В±Р С”Р В° Р С—РЎР‚Р С‘ РЎвЂЎРЎвЂљР ВµР Р…Р С‘Р С‘ РЎвЂћР В°Р в„–Р В»Р В°", err, nil)
	}
	src, err := file.Open()
	if err != nil {
		return nil, apperrors.ErrInternalServer
	}
	defer src.Close()
	if err := validation.ValidateFile(file, src, uploadContext); err != nil {
		return nil, apperrors.NewHttpError(http.StatusBadRequest, "Р В¤Р В°Р в„–Р В» Р Р…Р Вµ Р С—РЎР‚Р С•РЎв‚¬Р ВµР В» Р Р†Р В°Р В»Р С‘Р Т‘Р В°РЎвЂ Р С‘РЎР‹", err, nil)
	}
	rules, _ := config.UploadContexts[uploadContext]
	// Р вЂ”Р В°Р СР ВµР Р…РЎРЏР ВµР С c.fileStorage Р Р…Р В° ctrl.fileStorage
	savedPath, err := c.fileStorage.Save(src, file.Filename, rules.PathPrefix)
	if err != nil {
		return nil, apperrors.ErrInternalServer
	}
	fileURL := "/uploads/" + savedPath
	return &fileURL, nil
}
