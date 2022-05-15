package validator

import "regexp"

var HexRX = regexp.MustCompile("^#[0-9A-Fa-f]{6}$")

type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func Unique(vals []int64) bool {
	set := make(map[int64]struct{})

	for _, val := range vals {
		if _, ok := set[val]; ok {
			return false
		}
		set[val] = struct{}{}
	}
	return true
}
