package models

import (
	"context"

	"github.com/abibby/salusa/auth"
	"github.com/abibby/salusa/database/builder"
)

//go:generate spice generate:migration
type User struct {
	auth.EmailVerifiedUser

	// ID           int    `json:"id"       db:"id,primary,autoincrement"`
	// Username     string `json:"username" db:"username"`
	// Password     []byte `json:"-"        db:"-"`
	// PasswordHash []byte `json:"-"        db:"password"`
}

// var _ hooks.BeforeSaver = (*User)(nil)

func UserQuery(ctx context.Context) *builder.ModelBuilder[*User] {
	return builder.From[*User]().WithContext(ctx)
}

// func (u *User) BeforeSave(ctx context.Context, db database.DB) error {
// 	if u.Password != nil {
// 		log.Print("THIS IS JUST FOR AN EXAMPLE. REPLACE THIS")
// 		h := sha512.Sum512_256(u.Password)
// 		u.PasswordHash = h[:]
// 	}
// 	return nil
// }
