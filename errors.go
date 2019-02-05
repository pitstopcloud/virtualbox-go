package virtualbox

import (
	"fmt"
	"strings"
)

type OperationErrorType string

type OperationError struct {
	Path string
	// for e.g GET, PUT, WRITE, READ
	Op   string
	Type OperationErrorType
	Err  error
}

func (o OperationError) Error() string {
	return strings.Join([]string{o.Path, o.Op, string(o.Type), o.Err.Error()}, ", \n")
}

type ValidationError struct {
	Path string
	Err  error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}

type ValidationErrors struct {
	errors []ValidationError
}

func (v ValidationErrors) Error() string {
	var messages []string
	for _, v := range v.errors {
		messages = append(messages, v.Error())
	}

	return strings.Join(messages, "\n")
}
func (v ValidationErrors) Add(path string, err error) {
	v.errors = append(v.errors, ValidationError{path, err})
}

type AlreadyExists string

func (v AlreadyExists) Error() string {
	return string(v)
}

func (v AlreadyExists) New(item string, hints ...string) AlreadyExists {
	hint := ""
	if len(hints) > 0 {
		hint = strings.Join(append([]string{"Try the following: \n"}, hints...), "\n")
	}
	return AlreadyExists(fmt.Sprintf("%s already exists. %s", item, hint))
}

var AlreadyExistsErrorr AlreadyExists = "already exists"

func IsAlreadyExistsError(err error) bool {
	_, ok := err.(AlreadyExists)
	return ok
}

type AlreadyAttachedError string

func (v AlreadyAttachedError) Error() string {
	return string(v)
}
func IsAlreadyAttachedError(err error) bool {
	_, ok := err.(AlreadyAttachedError)
	return ok
}

type NotFoundError string

func (n NotFoundError) Error() string {
	return string(n)
}
