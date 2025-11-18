package lua

import _ "embed"

// ClaimScript contains the Redis Lua script for claims.
//
//go:embed claim.lua
var ClaimScript string
