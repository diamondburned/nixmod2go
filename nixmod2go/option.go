package nixmod2go

import "libdb.so/nixmod2go/nixmodule"

// eitherIsFloat returns true if the either type is actually the result of
// types.number. We want to handle this as a json.Number.
func eitherIsNumber(either nixmodule.EitherOption) bool {
	return len(either.Either) == 2 &&
		nixmodule.IsType[nixmodule.IntOption](either.Either[0]) &&
		nixmodule.IsType[nixmodule.FloatOption](either.Either[1])
}
