package middleware

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/services"
)

// ErrorHandler แปลง error ของ domain เป็นรหัสสถานะ HTTP ที่เหมาะสม
//
// ข้อความที่ส่งกลับเป็นภาษาไทยและบอกวิธีแก้ ส่วนรายละเอียดทางเทคนิคอยู่ใน log เท่านั้น
func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "เกิดข้อผิดพลาดภายในระบบ กรุณาลองใหม่อีกครั้ง"

		var fe *fiber.Error
		switch {
		case errors.As(err, &fe):
			status, message = fe.Code, fe.Message

		case errors.Is(err, domain.ErrInvalidCredentials):
			status, message = fiber.StatusUnauthorized, err.Error()
		case errors.Is(err, domain.ErrUserInactive):
			status, message = fiber.StatusForbidden, err.Error()
		case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrNotOwner):
			status, message = fiber.StatusForbidden, err.Error()
		case errors.Is(err, domain.ErrDatasetNotFound):
			status, message = fiber.StatusNotFound, err.Error()
		case errors.Is(err, domain.ErrDatasetNotReady):
			status, message = fiber.StatusConflict, err.Error()
		case errors.Is(err, domain.ErrPasswordTooShort):
			status, message = fiber.StatusBadRequest, err.Error()
		case errors.Is(err, services.ErrFileTooLarge), errors.Is(err, services.ErrNotExcelFile):
			status, message = fiber.StatusBadRequest, err.Error()
		}

		if status >= 500 {
			log.Error("request ล้มเหลว",
				"method", c.Method(), "path", c.Path(), "err", err)
		}
		return c.Status(status).JSON(fiber.Map{"error": message})
	}
}
