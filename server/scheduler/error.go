package scheduler

import (
	"errors"
)


var (
	ErrorJobNotFound =errors.New("job Not found")
	ErrorStreamNotAllowed =errors.New("upload not allowed")
	ErrorInvalidStatus =errors.New("job invalid status")
	ErrorFileSkipped=errors.New("path skipped")
)