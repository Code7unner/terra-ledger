package entity

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrDoublePledge = errors.New("active lien exists on parcel")
	ErrNotVerified  = errors.New("parcel not KYC verified")
	ErrUnauthorized = errors.New("invalid or missing API key")
)
