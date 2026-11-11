package users

type UpdateInput struct {
	RoleID      *string `json:"roleId"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phoneNumber"`
	BirthDate   *string `json:"birthDate"`
	Password    *string `json:"password"`
	FullName    *string `json:"fullName"`
	Gender      *string `json:"gender"`
	IsActive    *bool   `json:"isActive"`
}

type PasswordInput struct {
	ExistingPassword string `json:"exsitingPassword"`
	NewPassword      string `json:"newPassword"`
}
