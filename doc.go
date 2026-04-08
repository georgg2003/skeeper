// Skeeper is a password manager: a CLI keeps secrets encrypted locally and syncs ciphertext
// with a server. Two gRPC backends handle auth (accounts, JWTs) and vault sync; the wire
// format never carries your master password or plaintext—only encrypted payloads and salts.
//
// Binaries are under cmd/ (skeepercli, auther, skeeper). api/ is generated protobuf; pkg/
// has small shared helpers; internal/ holds the real service and client implementation.
package skeeper
