package utils

import "fmt"

// Success messages.
func CreateMessage(resource string) string {
	return fmt.Sprintf("🎉 %s created successfully! Welcome aboard! 🚀", resource)
}

func GetMessage(resource string) string {
	return fmt.Sprintf("📦 %s retrieved successfully! Smooth sailing! 😎", resource)
}

func UpdateMessage(resource string) string {
	return fmt.Sprintf("🔄 %s updated successfully! Looking fresh! ✨", resource)
}

func DeleteMessage(resource string) string {
	return fmt.Sprintf("🗑️ %s deleted successfully! Sayonara! 👋", resource)
}

func SetCookieMessage(cookieName string) string {
	return fmt.Sprintf("🍪 %s set successfully! Enjoy your stay! 😋", cookieName)
}

// Error messages.
func ErrorCreateMessage(resource string) string {
	return fmt.Sprintf("❌ Oops! Failed to create %s. Try again later! 🤦‍♂️", resource)
}

func ErrorGetMessage(resource string) string {
	return fmt.Sprintf("🚨 Whoops! Couldn't retrieve %s. Maybe it's playing hide and seek? 🤔", resource)
}

func ErrorUpdateMessage(resource string) string {
	return fmt.Sprintf("⚠️ Failed to update %s. Maybe it doesn't like change! 😅", resource)
}

func ErrorDeleteMessage(resource string) string {
	return fmt.Sprintf("🛑 Failed to delete %s. Looks like it's refusing to leave! 😭", resource)
}

func StrictDeleteMessage(resource string) string {
	return fmt.Sprintf("❌ Cannot delete %s! At least one %s is required to keep the system functional. 🔒", resource, resource)
}

func StrictAddMessage(resource string) string {
	return fmt.Sprintf("❌ Cannot create %s! %s already exists and cannot be duplicated. 🔒", resource, resource)
}

func AlreadyExistMessage(model, field string) string {
	if field != "" {
		model += " " + field
	}

	return fmt.Sprintf("⚡ %s already exists! No duplicates allowed! 😆", model)
}

func NotFoundMessage(model, field string) string {
	if field != "" {
		model += " " + field
	}

	return fmt.Sprintf("🔍 %s not found. Maybe it teleported to another dimension? 🛸", model)
}

func ValidationMessage(resource string) string {
	return fmt.Sprintf("🤷‍♂️ Validation failed for %s. Check your input and try again! 🔍", resource)
}

func AccessDeniedMessage(resource string) string {
	return fmt.Sprintf("🔒 Access Denied! Your %s does not have permission to access this resource. 🚫", resource)
}

func SessionExpiredMessage() string {
	return "⏳ Your session has expired. Please log in again to continue! 🔄"
}

func UploadErrorMessage(resource string) string {
	return fmt.Sprintf("📤 Oops! Failed to upload %s. Please try again later! 🚧", resource)
}

const (
	NotLoggedInMessage       = "🚪 You are not logged in. Please sign in to continue! 🔐"
	AlreadyLoggedInMessage   = "✅ You are already logged in. No need to log in again! 🎉"
	InvalidCredentialMessage = "🚫 Login failed. Please check your email and password, then try again! 🔍"
)
